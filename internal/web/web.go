package web

import (
	"bytes"
	"context"
	config "dev/go-keylight/internal"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/grandcat/zeroconf"
)

type Page struct {
	Data map[string]any
}

type KeylightState struct {
	NumberOfLights int `json:"numberOfLights"`
	Lights         []struct {
		IP          string
		On          int `json:"on"`
		Brightness  int `json:"brightness"`
		Temperature int `json:"temperature"`
	} `json:"lights"`
}

var templates = template.Must(template.ParseFiles("templates/index.html"))

// Fetches the current state of the light.
func getState(ip string, client *http.Client) (KeylightState, error) {
	var data KeylightState
	res, err := http.Get(fmt.Sprintf("http://%s:9123/elgato/lights", ip))
	if err != nil {
		fmt.Println("There has been an error polling the light.")
		return data, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("There has been an error reading the data received from the light.")
		return data, err
	}
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("There has been an error reading the data received from the light.")
		return data, err
	}
	data.Lights[0].IP = ip
	return data, nil
}

// Sends a request to the Elgato device.
func sendRequest(body []byte, url string, client *http.Client) error {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

// Returns the index page of the site.
func indexHandler(w http.ResponseWriter, r *http.Request, settings *config.Config) {
	// Return 404 if not accessing root
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	ips := settings.IPs

	var data []KeylightState
	for _, v := range ips {
		if v == "" {
			fmt.Println("No valid IP to poll.")
			continue
		}
		res, err := getState(v, &http.Client{})
		if err != nil {
			fmt.Println("There has been an error polling the light.")
			continue
		}
		data = append(data, res)
	}

	renderPage(w, &Page{Data: map[string]any{"state": data}})
}

// Discovers any elgato devices connected to the local network.
func getHandler(w http.ResponseWriter, r *http.Request, settings *config.Config) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	var data []zeroconf.ServiceEntry
	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			entry.Instance = strings.ReplaceAll(entry.Instance, "\\", "")
			settings.SetIP((fmt.Sprint(entry.AddrIPv4[0])))
			data = append(data, *entry)
		}
		log.Println("No more entries.")
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	err = resolver.Browse(ctx, "_elg._tcp", "local.", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	<-ctx.Done()
	templates.ExecuteTemplate(w, "newlights", &Page{Data: map[string]any{"lights": data}})
}

// Receives a response to toggle the light on or off.
func onHandler(w http.ResponseWriter, r *http.Request, client *http.Client, state KeylightState) {
	var response []byte
	if state.Lights[0].On == 0 {
		response = []byte("Off")
	} else {
		response = []byte("On")
	}
	body := []byte(fmt.Sprintf(`{"lights": [{ "on": %d}]}`, 1-state.Lights[0].On))
	url := fmt.Sprintf("http://%s:9123/elgato/lights", state.Lights[0].IP)
	if err := sendRequest(body, url, client); err != nil {
		fmt.Println("There's been an error sending a request to the associated keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(response)
}

// Receives a response for brightness change.
func brightnessHandler(w http.ResponseWriter, r *http.Request, client *http.Client, state KeylightState) {
	brightness := r.FormValue("brightness")
	body := []byte(fmt.Sprintf(`{"lights": [{ "brightness": %s}]}`, brightness))
	url := fmt.Sprintf("http://%s:9123/elgato/lights", state.Lights[0].IP)
	if err := sendRequest(body, url, client); err != nil {
		fmt.Println("There's been an error sending a request to the associated keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Receives a response for temperature change.
func temperatureHandler(w http.ResponseWriter, r *http.Request, client *http.Client, state KeylightState) {
	temperature := r.FormValue("temperature")
	body := []byte(fmt.Sprintf(`{"lights": [{ "temperature": %s}]}`, temperature))
	url := fmt.Sprintf("http://%s:9123/elgato/lights", state.Lights[0].IP)
	if err := sendRequest(body, url, client); err != nil {
		fmt.Println("There's been an error sending a request to the associated keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Closure for handler function that allows for necessary config to be included in parameters.
func stateChangeClosure(fn func(http.ResponseWriter, *http.Request, *http.Client, KeylightState)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.FormValue("ip")
		client := &http.Client{}
		state, err := getState(ip, client)
		if err != nil {
			fmt.Println("There's been an error polling the keylight.")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fn(w, r, client, state)
	}
}

// Closure for handler function that allows for necessary config to be included in parameters.
func settingsClosure(fn func(http.ResponseWriter, *http.Request, *config.Config), settings *config.Config) http.HandlerFunc {
	settings = settings.InitConfig()
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, settings)
	}
}

// Renders the page with the selected template.
func renderPage(w http.ResponseWriter, page *Page) {
	var dep = `<link href="https://cdn.jsdelivr.net/npm/beercss@3.4.9/dist/cdn/beer.min.css" rel="stylesheet">
<script type="module" src="https://cdn.jsdelivr.net/npm/beercss@3.4.9/dist/cdn/beer.min.js"></script>
<script type="module" src="https://cdn.jsdelivr.net/npm/material-dynamic-colors@1.1.0/dist/cdn/material-dynamic-colors.min.js"></script>
<script src="https://unpkg.com/htmx.org@1.9.9"></script>`

	page.Data["dep"] = dep
	err := templates.ExecuteTemplate(w, "index.html", page)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func Start(port string) {
	settings := &config.Config{}
	//Handlers
	http.HandleFunc("/get", settingsClosure(getHandler, settings))
	http.HandleFunc("/", settingsClosure(indexHandler, settings))
	http.HandleFunc("/on", stateChangeClosure(onHandler))
	http.HandleFunc("/brightness", stateChangeClosure(brightnessHandler))
	http.HandleFunc("/temperature", stateChangeClosure(temperatureHandler))
	//Start Server
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
