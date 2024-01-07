package keylight

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
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

func Discover() []zeroconf.ServiceEntry {
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

	return data
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
