package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	cfgFile := flag.String("c", "/data/options.json", "configuration file")
	flag.Usage = usage
	flag.Parse()

	Logger = log.Default()

	logf("Reading configuration")
	cfg, err := ParseConfigFile(*cfgFile)
	if err != nil {
		fatalf("Configuration error: %v", err)
	}

	logf("Initializing HassShooter")
	hs, err := NewHassShooter(cfg)
	if err != nil {
		fatalf("Could not create a new HassShooter: %v", err)
	}
	defer hs.Close()

	logf("Serving HTTP requests")
	if err := hs.ListenAndServe(); err != nil {
		logf("Server exited: %v", err)
	}
}

// usage prints the command usage message and flags.
func usage() {
	fmt.Fprintf(os.Stderr, "usage: hass-shooter [flags]\n")
	flag.PrintDefaults()
}
