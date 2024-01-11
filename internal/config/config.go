package config

import (
	"dev/go-keylight/internal/keylight"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Config struct {
	json     JSONConfig
	LightMap map[string]*Lights
}

type Lights struct {
	IP   string `json:"ip"`
	Name string `json:"name"`
	KeepAwake
}

type KeepAwake struct {
	On     bool `json:"keepAwake"`
	ticker *time.Ticker
}

type JSONConfig map[string]Lights

// Config Setter/Getter

func (c *Config) AddLight(ip string, name string, ka bool) {
	c.json[ip] = Lights{IP: ip, Name: name, KeepAwake: KeepAwake{On: ka}}
	jsonFile, err := os.Create("config.json")
	if err != nil {
		fmt.Println("Failure to open config.json")
		return
	}
	enc := json.NewEncoder(jsonFile)
	if err := enc.Encode(&c.json); err != nil {
		log.Println(err)
	}
	c.LightMap[ip] = &Lights{IP: ip, Name: name, KeepAwake: KeepAwake{On: ka}}
}

func InitConfig() *Config {
	c := &Config{json: nil, LightMap: map[string]*Lights{}}
	if c.json == nil {
		j := JSONConfig{}
		file, err := os.Open("config.json")
		if err != nil {
			fmt.Println("Failure to open config.json, creating file.")
			os.WriteFile("config.json", nil, 0600)
			c.json = j
			return c
		}
		dec := json.NewDecoder(file)
		if err := dec.Decode(&j); err != nil {
			log.Println(err)
		}
		c.json = j
		for _, v := range c.json {
			c.LightMap[v.IP] = &Lights{IP: v.IP, Name: v.Name, KeepAwake: v.KeepAwake}
		}
	}
	return c
}

func (c *Config) KeepAwake(client *http.Client) {
	for i, light := range c.LightMap {
		if light.KeepAwake.On {
			ip := light.IP
			c.LightMap[i].KeepAwake = KeepAwake{On: true, ticker: time.NewTicker(1 * time.Hour)}
			go func() {
				for {
					select {
					case <-c.LightMap[ip].ticker.C:
						keylight.GetState(ip, client)
					}
				}
			}()
		}
	}
}

func (c *Config) DisableKeepAwake(ip string) {
	if c.LightMap[ip].ticker.C != nil {
		c.LightMap[ip].ticker.Stop()
		c.AddLight(c.LightMap[ip].IP, c.LightMap[ip].Name, false)
		return
	}
}
