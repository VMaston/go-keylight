package keylight

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type KeylightState struct {
	NumberOfLights int `json:"numberOfLights"`
	Lights         []struct {
		IP          string
		On          int `json:"on"`
		Brightness  int `json:"brightness"`
		Temperature int `json:"temperature"`
	} `json:"lights"`
}

// Fetches the current state of the light.
func GetState(ip string, client *http.Client) (KeylightState, error) {
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
func SendRequest(body []byte, url string, client *http.Client) error {
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
