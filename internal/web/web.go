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
	fmt.Println(data)
	return data, nil
}

func indexHandler(w http.ResponseWriter, r *http.Request, ips []string) {
	// Return 404 if not accessing root
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	renderPage(w, &Page{Data: map[string]any{"ips": ips}})
}

func onHandler(w http.ResponseWriter, r *http.Request) {
	ip := r.FormValue("ip")
	client := &http.Client{}
	state, err := getState(ip, client)
	if err != nil {
		fmt.Println("There's been an error polling the keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var onState int
	var response []byte
	if state.Lights[0].On == 0 {
		onState = 1
		response = []byte("Off")
	} else {
		onState = 0
		response = []byte("On")
	}
	body := []byte(fmt.Sprintf(`{"lights": [{ "on": %d}]}`, onState))
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://%s:9123/elgato/lights", ip), bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("There's been an error sending a request to the associated keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("There's been an error sending a request to the associated keylight.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(response)
	defer res.Body.Close()
}

// Closure for handler function that allows for necessary config to be included in parameters.
func handlerHandler(fn func(http.ResponseWriter, *http.Request, []string), ips []string) http.HandlerFunc {
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
	http.HandleFunc("/", handlerHandler(indexHandler, ips))
	http.HandleFunc("/on", onHandler)
	//Start Server
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
