package web

import (
	"dev/go-keylight/internal/config"
	"dev/go-keylight/internal/keylight"
	"fmt"
	"log"
	"net/http"
	"text/template"
)

// Page is a struct that contains a key:value map for inserting untyped data into a HTML template.
type Page struct {
	Data map[string]any
}

var templates = template.Must(template.ParseFiles("templates/index.html"))

// indexHandler serves the index page of the site, denying requests to pages other than root.
// Its main purpose is to call keylight.GetState() to poll the lights for their current settings and render that data into the template.
func indexHandler(w http.ResponseWriter, r *http.Request, client *http.Client, settings *config.Config) {
	// Return 404 if not accessing root
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	lights := settings.LightMap

	var data []keylight.Keylight
	for _, v := range lights {
		if v.IP == "" {
			fmt.Println("No valid IP to poll.")
			continue
		}
		res, err := keylight.GetState(v.IP, client)
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
func discoverHandler(w http.ResponseWriter, r *http.Request, client *http.Client, settings *config.Config) {
	data := keylight.Discover()
	templates.ExecuteTemplate(w, "newlights", &Page{Data: map[string]any{"lights": data}})
}

// addHandler calls the settings.SetIP function to append the IPs from the request to the config file and updates the settings object, then refreshes the page.
func addHandler(w http.ResponseWriter, r *http.Request, client *http.Client, settings *config.Config) {
	r.ParseForm()
	for i, v := range r.Form["ip"] {
		settings.AddLight(v[1:len(v)-1], r.Form["name"][i], false)
	}
	w.Header().Add("HX-Refresh", "true")
}

func keepAwakeHandler(w http.ResponseWriter, r *http.Request, client *http.Client, settings *config.Config) {
	ip := r.FormValue("ip")
	slider := r.FormValue("switch-" + ip)
	if slider == "" {
		settings.DisableKeepAwake(ip)
	} else if slider == "on" {
		settings.LightMap[ip].KeepAwake.On = true
		settings.AddLight(settings.LightMap[ip].IP, settings.LightMap[ip].Name, true)
		settings.KeepAwake(client)
	}
}

// onHandler calls keylight.SendRequest to toggle the light on or off depending on the current state of the light and returns the opposite toggle for the button.
func onHandler(w http.ResponseWriter, r *http.Request, client *http.Client, state keylight.Keylight, url string) {
	var response []byte
	if state.On == 0 {
		response = []byte("Off")
	} else {
		response = []byte("On")
	}
	body := []byte(fmt.Sprintf(`{"lights": [{ "on": %d}]}`, 1-state.On))

	if err := keylight.SendRequest(body, url, client); err != nil {
		fmt.Println("There's been an error sending a request to the associated keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(response)
}

// brightnessHandler calls keylight.SendRequest to adjust the brightness value based on the user input from the frontend.
func brightnessHandler(w http.ResponseWriter, r *http.Request, client *http.Client, state keylight.Keylight, url string) {
	brightness := r.FormValue("brightness")
	body := []byte(fmt.Sprintf(`{"lights": [{ "brightness": %s}]}`, brightness))

	if err := keylight.SendRequest(body, url, client); err != nil {
		fmt.Println("There's been an error sending a request to the associated keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// temperatureHandler calls keylight.SendRequest to adjust the temperature value based on the user input from the frontend.
func temperatureHandler(w http.ResponseWriter, r *http.Request, client *http.Client, state keylight.Keylight, url string) {
	temperature := r.FormValue("temperature")
	body := []byte(fmt.Sprintf(`{"lights": [{ "temperature": %s}]}`, temperature))

	if err := keylight.SendRequest(body, url, client); err != nil {
		fmt.Println("There's been an error sending a request to the associated keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// stateChangeClosure initializes a HTTP client and calls keylight.GetState for the initial states when using on, brightness or temperature adjustment handlers.
func stateChangeClosure(fn func(http.ResponseWriter, *http.Request, *http.Client, keylight.Keylight, string), client *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.FormValue("ip")
		state, err := keylight.GetState(ip, client)
		url := fmt.Sprintf("http://%s:9123/elgato/lights", state.IP)
		if err != nil {
			fmt.Println("There's been an error polling the keylight.")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fn(w, r, client, state, url)
	}
}

// settingsClosure intializes the settings object (shared between handlers via pointer)
func settingsClosure(fn func(http.ResponseWriter, *http.Request, *http.Client, *config.Config), client *http.Client, settings *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, client, settings)
	}
}

// renderPage renders the page using ExecuteTemplate while inserting the Data map contained in the Page struct.
func renderPage(w http.ResponseWriter, page *Page) {
	err := templates.ExecuteTemplate(w, "index.html", page)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Start initializes the config, registers the handlers and starts the webserver.
func Start(port string) {
	client := &http.Client{}
	settings := config.InitConfig()
	settings.KeepAwake(client)

	//Handlers
	http.HandleFunc("/discover", settingsClosure(discoverHandler, client, settings))
	http.HandleFunc("/add", settingsClosure(addHandler, client, settings))
	http.HandleFunc("/", settingsClosure(indexHandler, client, settings))
	http.HandleFunc("/on", stateChangeClosure(onHandler, client))
	http.HandleFunc("/brightness", stateChangeClosure(brightnessHandler, client))
	http.HandleFunc("/temperature", stateChangeClosure(temperatureHandler, client))
	http.HandleFunc("/keepawake", settingsClosure(keepAwakeHandler, client, settings))
	//Start Server
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
