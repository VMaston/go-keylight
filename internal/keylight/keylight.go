package keylight

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Keylight struct {
	Name        string
	IP          string
	On          int
	Brightness  int
	Temperature int
	KeepAwake   bool
}

type KeylightJSON struct {
	NumberOfLights int `json:"numberOfLights"`
	Lights         []struct {
		On          int `json:"on"`
		Brightness  int `json:"brightness"`
		Temperature int `json:"temperature"`
	} `json:"lights"`
}

// Fetches the current state of the light.
func GetState(ip string, client *http.Client) (Keylight, error) {
	var data KeylightJSON
	var keylight Keylight
	res, err := http.Get(fmt.Sprintf("http://%s:9123/elgato/lights", ip))
	if err != nil {
		fmt.Println("There has been an error polling the light.")
		return keylight, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("There has been an error reading the data received from the light.")
		return keylight, err
	}
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("There has been an error reading the data received from the light.")
		return keylight, err
	}
	keylight.IP = ip
	keylight.Brightness = data.Lights[0].Brightness
	keylight.Temperature = data.Lights[0].Temperature
	keylight.On = data.Lights[0].On

	return keylight, nil
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
