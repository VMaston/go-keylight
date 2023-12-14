package main

import (
	"dev/go-keylight/internal/web"
	"fmt"
	"os"
	"strings"
)

func main() {
	config, err := os.ReadFile("config.txt")
	if err != nil {
		fmt.Println("Failure to read file, aborting...")
		panic(err)
	}
	ips := strings.Split(string(config), "\n")
	web.Start("8080", ips)
}
