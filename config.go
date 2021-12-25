package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Config contains the HassShooter configuration.
type Config struct {
	HassBaseURL     string `json:"hass_base_url"`
	HassToken       string `json:"hass_token"`
	HassPages       []Page `json:"hass_pages"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	Rotation        int    `json:"rotation"`
	ListenAddr      string `json:"listen_addr"`
	RefreshTimeSecs int    `json:"refresh_time"`
	MinIdleTimeSecs int    `json:"min_idle_time"`
	TimeoutSecs     int    `json:"timeout"`
}

// Page represents a page to be captured.
type Page struct {
	// Path is the URL path of the page.
	Path string `json:"path"`

	// Scale is the scale factor used to take the screenshot.
	Scale float64 `json:"scale"`
}

// ParseConfig creates a new Config from the specified configuration file.
func ParseConfigFile(cfgFile string) (Config, error) {
	f, err := os.Open(cfgFile)
	if err != nil {
		return Config{}, fmt.Errorf("could not open file: %w", err)
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("could not decode config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration. It returns error if the configuration
// is not valid.
func (cfg Config) Validate() error {
	if cfg.HassBaseURL == "" {
		return errors.New("HASS base URL is missing")
	}

	if cfg.HassToken == "" {
		return errors.New("HASS token is missing")
	}

	if len(cfg.HassPages) == 0 {
		return errors.New("no pages to capture")
	}

	if cfg.Width == 0 || cfg.Height == 0 {
		return errors.New("width and height cannot be 0")
	}

	if cfg.ListenAddr == "" {
		return errors.New("listen address is missing")
	}

	return nil
}
