package config

import (
	"dev/go-keylight/internal/keylight"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// TODO: Make []Lights a map[string]Lights type to avoid having to iterate-search for the IP.
type Config struct {
	data   []byte
	Lights []Lights
}

type Lights struct {
	IP   string
	Name string
	KeepAwake
}

type KeepAwake struct {
	On     bool
	ticker *time.Ticker
}

// Config Setter/Getter

func (c *Config) AddLight(ip string, name string) {
	line := []byte("\n" + ip + ", " + name + ", " + "false")
	c.data = append(c.data, line...)
	err := os.WriteFile("config.txt", c.data, 0600)
	if err != nil {
		fmt.Println("Failure to write to file, aborting...")
		panic(err)
	}
	c.Lights = append(c.Lights, Lights{IP: ip, Name: name, KeepAwake: KeepAwake{On: false}})
}

func InitConfig() *Config {
	c := &Config{}
	if c.data == nil {
		file, err := os.ReadFile("config.txt")
		if err != nil {
			fmt.Println("Failure to read file, creating new config.txt...")
			os.WriteFile("config.txt", nil, 0600)
		}
		c.data = file
		lines := strings.Split(string(file), "\n")
		for _, v := range lines {
			fields := strings.Split(v, ", ")
			ka, err := strconv.ParseBool(fields[2])
			if err != nil {
				fmt.Println("Failure to convert config file KeepAwake to boolean value. Setting false as default.")
				ka = false
			}
			c.Lights = append(c.Lights, Lights{IP: fields[0], Name: fields[1], KeepAwake: KeepAwake{On: ka}})
		}
	}
	return c
}

func (c *Config) KeepAwake(client *http.Client) {
	for i, light := range c.Lights {
		if light.KeepAwake.On {
			IP := light.IP
			c.Lights[i].KeepAwake = KeepAwake{On: true, ticker: time.NewTicker(1 * time.Hour)}
			idx := i
			go func() {
				for {
					select {
					case <-c.Lights[idx].ticker.C:
						keylight.GetState(IP, client)
					}
				}
			}()
		}
	}
}

func (c *Config) DisableKeepAwake(ip string) {
	for i, light := range c.Lights {
		if light.IP == ip {
			if light.KeepAwake.ticker.C != nil {
				idx := i
				c.Lights[idx].ticker.Stop()
				return
			}
		}
	}
}
