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
	data     []byte
	LightMap map[string]*Lights
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
	c.LightMap[ip] = &Lights{IP: ip, Name: name, KeepAwake: KeepAwake{On: false}}
}

func InitConfig() *Config {
	c := &Config{data: nil, LightMap: map[string]*Lights{}}
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
			c.LightMap[fields[0]] = &Lights{IP: fields[0], Name: fields[1], KeepAwake: KeepAwake{On: ka}}
		}
	}
	return c
}

// TODO: Write KeepAwake status to file.
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
		return
	}
}
