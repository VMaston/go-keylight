package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	data []byte
	IPs  []string
}

// Config Setter/Getter

func (c *Config) SetIP(ip string) *Config {
	c.data = append(c.data, "\n"+ip...)
	err := os.WriteFile("config.txt", c.data, 0600)
	if err != nil {
		fmt.Println("Failure to write to file, aborting...")
		panic(err)
	}
	c.IPs = append(c.IPs, ip)
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
		c.IPs = strings.Fields(string(file))
	}
	return c
}
