package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"text/template"
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
	fmt.Println(data)
	return data, nil
}

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

func indexHandler(w http.ResponseWriter, r *http.Request, ips []string) {
	// Return 404 if not accessing root
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	var data []KeylightState
	for _, v := range ips {
		res, err := getState(v, &http.Client{})
		if err != nil {
			fmt.Println("There has been an error polling the light.")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data = append(data, res)
	}

	renderPage(w, &Page{Data: map[string]any{"state": data}})
}

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

func brightnessHandler(w http.ResponseWriter, r *http.Request, client *http.Client, state KeylightState) {
	brightness := r.FormValue("brightness")
	body := []byte(fmt.Sprintf(`{"lights": [{ "brightness": %s}]}`, brightness))
	url := fmt.Sprintf("http://%s:9123/elgato/lights", state.Lights[0].IP)
	if err := sendRequest(body, url, client); err != nil {
		fmt.Println("There's been an error sending a request to the associated keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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
func indexClosure(fn func(http.ResponseWriter, *http.Request, []string), ips []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, ips)
	}
}

// Renders the page with the selected template.
func renderPage(w http.ResponseWriter, page *Page) {

	var dep = `<link href="https://cdn.jsdelivr.net/npm/beercss@3.4.9/dist/cdn/beer.min.css" rel="stylesheet">
<script type="module" src="https://cdn.jsdelivr.net/npm/beercss@3.4.9/dist/cdn/beer.min.js"></script>
<script type="module" src="https://cdn.jsdelivr.net/npm/material-dynamic-colors@1.1.0/dist/cdn/material-dynamic-colors.min.js"></script>
<script src="https://unpkg.com/htmx.org@1.9.9"></script>`

	page.Data["dep"] = dep
	var templates = template.Must(template.ParseFiles("templates/index.html"))
	err := templates.ExecuteTemplate(w, "index.html", page)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func Start(port string, ips []string) {
	//Handlers
	http.HandleFunc("/", indexClosure(indexHandler, ips))
	http.HandleFunc("/on", stateChangeClosure(onHandler))
	http.HandleFunc("/brightness", stateChangeClosure(brightnessHandler))
	http.HandleFunc("/temperature", stateChangeClosure(temperatureHandler))
	//Start Server
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
