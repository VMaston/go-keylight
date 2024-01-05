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

type Config struct {
	data   []byte
	Lights []struct {
		IP        string
		Name      string
		KeepAwake bool
	}
}

// Config Setter/Getter

func (c *Config) AddLight(ip string, name string) *Config {
	line := []byte("\n" + ip + ", " + name + ", " + "false")
	c.data = append(c.data, line...)
	err := os.WriteFile("config.txt", c.data, 0600)
	if err != nil {
		fmt.Println("Failure to write to file, aborting...")
		panic(err)
	}
	c.Lights = append(c.Lights, struct {
		IP        string
		Name      string
		KeepAwake bool
	}{IP: ip, Name: name, KeepAwake: false})
	return c
}

func (c *Config) InitConfig() *Config {
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
			c.Lights = append(c.Lights, struct {
				IP        string
				Name      string
				KeepAwake bool
			}{IP: fields[0], Name: fields[1], KeepAwake: ka})
		}
	}
	return c
}

func (c *Config) KeepAwake(client *http.Client) {
	for _, light := range c.Lights {
		if light.KeepAwake {
			IP := light.IP
			ticker := time.NewTicker(1 * time.Second)
			quit := make(chan struct{})

			go func() {
				for {
					select {
					case <-ticker.C:
						keylight.GetState(IP, client)
					case <-quit:
						ticker.Stop()
						return
					}
				}
			}()
		}
	}
}
