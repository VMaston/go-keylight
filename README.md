# go-keylight
> An elgato keylight controller written in Golang.

Created with [Golang](https://github.com/golang/go), [HTMX](https://github.com/bigskysoftware/htmx) & [BeerCSS](https://github.com/beercss/beercss)

## Features

### Discovery/Connection

go-keylight uses the [ZeroConf mDNS](https://github.com/grandcat/zeroconf) libary to scan, authenticate and connect with available keylights.
Keylight data is stored in a config.json file that is created upon the program's first launch.

![Discovery](https://i.imgur.com/XtItqQb.png)

### Adjustment

- On/Off State
- Brightness
- Temperature

![Adjustment](https://i.imgur.com/ZBTwVv4.png)

#### Keep Awake

For those of us who have had issues with a device going into sleep mode and deauthenticating from the router, this program has a toggle to ping the device every hour.

#### Removal

Existing keylights can be removed in a simple dropdown.

![Removal](https://i.imgur.com/HbDw9Fg.png)

===

Runs on port 8080. Can be changed in cmd/main.go, line 8 if you wanted to run on a different port.

Made public as a portfolio piece for my knowledge of Golang.
