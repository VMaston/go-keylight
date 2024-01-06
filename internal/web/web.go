package web

import (
	"context"
	"dev/go-keylight/internal/config"
	"dev/go-keylight/internal/keylight"
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/grandcat/zeroconf"
)

// Page is a struct that contains a key:value map for inserting untyped data into a HTML template.
type Page struct {
	Data map[string]any
}

var templates = template.Must(template.ParseFiles("templates/index.html"))

// indexHandler serves the index page of the site, denying requests to pages other than root.
// Its main purpose is to call keylight.GetState() to poll the lights for their current settings and render that data into the template.
func indexHandler(w http.ResponseWriter, r *http.Request, settings *config.Config) {
	// Return 404 if not accessing root
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	lights := settings.Lights

	var data []keylight.Keylight
	for _, v := range lights {
		if v.IP == "" {
			fmt.Println("No valid IP to poll.")
			continue
		}
		res, err := keylight.GetState(v.IP, &http.Client{})
		if err != nil {
			fmt.Println("There has been an error polling the light.")
			continue
		}
		res.Name = v.Name
		res.KeepAwake = v.KeepAwake.On
		data = append(data, res)
	}

	renderPage(w, &Page{Data: map[string]any{"state": data}})
}

// discoverHandler uses the zeroconf package to discover any elgato devices connected to the local network and returns that data to the html template.
func discoverHandler(w http.ResponseWriter, r *http.Request, settings *config.Config) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	var data []zeroconf.ServiceEntry
	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			entry.Instance = strings.ReplaceAll(entry.Instance, "\\", "")
			data = append(data, *entry)
		}
		log.Println("No more entries.")
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err = resolver.Browse(ctx, "_elg._tcp", "local.", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	<-ctx.Done()
	templates.ExecuteTemplate(w, "newlights", &Page{Data: map[string]any{"lights": data}})
}

// addHandler calls the settings.SetIP function to append the IPs from the request to the config file and updates the settings object, then refreshes the page.
func addHandler(w http.ResponseWriter, r *http.Request, settings *config.Config) {
	r.ParseForm()
	fmt.Println(r.Form)
	for i, v := range r.Form["ip"] {
		settings.AddLight(v[1:len(v)-1], r.Form["name"][i])
	}
	w.Header().Add("HX-Refresh", "true")
}

func keepAwakeHandler(w http.ResponseWriter, r *http.Request, settings *config.Config) {
	ip := r.FormValue("ip")
	slider := r.FormValue("switch-" + ip)
	if slider == "off" {
		settings.DisableKeepAwake(ip)
	} else if slider == "on" {
		fmt.Println("HELLO")
		for i, light := range settings.Lights {
			if light.IP == ip {
				settings.Lights[i].KeepAwake.On = true
			}
		}
		settings.KeepAwake(&http.Client{})
	}
}

// onHandler calls keylight.SendRequest to toggle the light on or off depending on the current state of the light and returns the opposite toggle for the button.
func onHandler(w http.ResponseWriter, r *http.Request, client *http.Client, state keylight.Keylight) {
	var response []byte
	if state.On == 0 {
		response = []byte("Off")
	} else {
		response = []byte("On")
	}
	body := []byte(fmt.Sprintf(`{"lights": [{ "on": %d}]}`, 1-state.On))
	url := fmt.Sprintf("http://%s:9123/elgato/lights", state.IP)
	if err := keylight.SendRequest(body, url, client); err != nil {
		fmt.Println("There's been an error sending a request to the associated keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(response)
}

// brightnessHandler calls keylight.SendRequest to adjust the brightness value based on the user input from the frontend.
func brightnessHandler(w http.ResponseWriter, r *http.Request, client *http.Client, state keylight.Keylight) {
	brightness := r.FormValue("brightness")
	body := []byte(fmt.Sprintf(`{"lights": [{ "brightness": %s}]}`, brightness))
	url := fmt.Sprintf("http://%s:9123/elgato/lights", state.IP)
	if err := keylight.SendRequest(body, url, client); err != nil {
		fmt.Println("There's been an error sending a request to the associated keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// temperatureHandler calls keylight.SendRequest to adjust the temperature value based on the user input from the frontend.
func temperatureHandler(w http.ResponseWriter, r *http.Request, client *http.Client, state keylight.Keylight) {
	temperature := r.FormValue("temperature")
	body := []byte(fmt.Sprintf(`{"lights": [{ "temperature": %s}]}`, temperature))
	url := fmt.Sprintf("http://%s:9123/elgato/lights", state.IP)
	if err := keylight.SendRequest(body, url, client); err != nil {
		fmt.Println("There's been an error sending a request to the associated keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// stateChangeClosure initializes a HTTP client and calls keylight.GetState for the initial states when using on, brightness or temperature adjustment handlers.
func stateChangeClosure(fn func(http.ResponseWriter, *http.Request, *http.Client, keylight.Keylight), client *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.FormValue("ip")
		state, err := keylight.GetState(ip, client)
		if err != nil {
			fmt.Println("There's been an error polling the keylight.")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fn(w, r, client, state)
	}
}

// settingsClosure intializes the settings object (shared between handlers via pointer)
func settingsClosure(fn func(http.ResponseWriter, *http.Request, *config.Config), settings *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, settings)
	}
}

// renderPage renders the page using ExecuteTemplate while inserting the Data map contained in the Page struct.
func renderPage(w http.ResponseWriter, page *Page) {
	var dep = `<link href="https://cdn.jsdelivr.net/npm/beercss@3.4.11/dist/cdn/beer.min.css" rel="stylesheet">
	<script type="module" src="https://cdn.jsdelivr.net/npm/beercss@3.4.11/dist/cdn/beer.min.js"></script>
	<script type="module" src="https://cdn.jsdelivr.net/npm/material-dynamic-colors@1.1.0/dist/cdn/material-dynamic-colors.min.js"></script>
	<script src="https://unpkg.com/htmx.org@1.9.9"></script>`

	page.Data["dep"] = dep
	err := templates.ExecuteTemplate(w, "index.html", page)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Start initializes the config, registers the handlers and starts the webserver.
func Start(port string) {
	client := &http.Client{}
	settings := &config.Config{}
	settings = settings.InitConfig()
	settings.KeepAwake(client)

	//Handlers
	http.HandleFunc("/discover", settingsClosure(discoverHandler, settings))
	http.HandleFunc("/add", settingsClosure(addHandler, settings))
	http.HandleFunc("/", settingsClosure(indexHandler, settings))
	http.HandleFunc("/on", stateChangeClosure(onHandler, client))
	http.HandleFunc("/brightness", stateChangeClosure(brightnessHandler, client))
	http.HandleFunc("/temperature", stateChangeClosure(temperatureHandler, client))
	http.HandleFunc("/keepawake", settingsClosure(keepAwakeHandler, settings))
	//Start Server
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
