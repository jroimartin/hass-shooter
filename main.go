package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type config struct {
	hassBaseURL string
	hassToken   string
	hassPaths   []string
	scales      []float64
	width       int
	height      int
	listenAddr  string
}

const (
	idleTime    = 1 * time.Second
	pageTimeout = 10 * time.Second
)

var (
	browser *rod.Browser

	cfg config
)

func main() {
	var err error

	log.Print("Reading config from environment")
	cfg, err = readConfig()
	if err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	log.Print("Initializing browser")
	browser, err = initBrowser()
	if err != nil {
		log.Fatalf("Could not initialize browser: %v", err)
	}
	defer browser.MustClose()

	log.Print("Login into HASS")
	if err := hassLogin(); err != nil {
		log.Fatalf("set-up error: %v", err)
	}

	http.HandleFunc("/", handler)

	log.Println("Serving HTTP requests")
	log.Fatal(http.ListenAndServe(cfg.listenAddr, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v - %v - %v %v", r.RemoteAddr, r.UserAgent(), r.Method, r.URL)

	var idx int
	if r.URL.Path == "/" {
		idx = 0
	} else {
		var err error
		idx, err = strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/"))
		if err != nil {
			log.Printf("Could not parse path index (%v)", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if idx < 0 || idx >= len(cfg.hassPaths) || (len(cfg.scales) != 1 && idx >= len(cfg.scales)) {
		log.Printf("Index is out of bounds (%v)", idx)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	path := cfg.hassPaths[idx]

	var scale float64
	if len(cfg.scales) == 1 {
		scale = cfg.scales[0]
	} else {
		scale = cfg.scales[idx]
	}

	log.Printf("Taking screenshot (path=%v; scale=%v)", path, scale)
	img, err := screenshot(path, scale)
	if err != nil {
		log.Printf("Could not take screenshot: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Print("Transforming image")
	img, err = transform(img)
	if err != nil {
		log.Printf("Could not transform image: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(w, bytes.NewReader(img)); err != nil {
		log.Printf("Could not write image: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func screenshot(path string, scale float64) ([]byte, error) {
	url := cfg.hassBaseURL + path
	page, err := browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return nil, fmt.Errorf("could not open page: %w", err)
	}
	defer page.MustClose()

	metrics := &proto.EmulationSetDeviceMetricsOverride{
		Width:             int(float64(cfg.width) / scale),
		Height:            int(float64(cfg.height) / scale),
		DeviceScaleFactor: scale,
	}
	if err := page.SetViewport(metrics); err != nil {
		return nil, fmt.Errorf("could not set window size: %w", err)
	}

	waitFcn := page.Timeout(pageTimeout).WaitRequestIdle(idleTime, nil, nil)
	waitFcn()

	img, err := page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
	})
	if err != nil {
		return nil, fmt.Errorf("could not take screenshot: %w", err)
	}

	return img, nil
}

func transform(img []byte) ([]byte, error) {
	cmd := exec.Command(
		"convert",
		"png:-",
		"-monochrome",
		"bmp:-",
	)
	cmd.Stdin = bytes.NewReader(img)
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("imagemagick error: %w", err)
	}

	return buf.Bytes(), nil
}

func hassLogin() error {
	page, err := browser.Page(proto.TargetCreateTarget{URL: cfg.hassBaseURL})
	if err != nil {
		return fmt.Errorf("could not open page: %w", err)
	}
	defer page.MustClose()

	hassTokens := struct {
		HASSURL     string `json:"hassUrl"`
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}{
		HASSURL:     cfg.hassBaseURL,
		AccessToken: cfg.hassToken,
		TokenType:   "Bearer",
	}
	hassTokensData, err := json.Marshal(hassTokens)
	if err != nil {
		return fmt.Errorf("could not marshal HASS token struct: %w", err)
	}

	_, err = page.Eval(
		`localStorage.setItem("hassTokens", arguments[0]);`,
		[]interface{}{string(hassTokensData)},
	)
	if err != nil {
		return fmt.Errorf("could not execute script: %w", err)
	}

	return nil
}

func initBrowser() (*rod.Browser, error) {
	path, ok := launcher.LookPath()
	if !ok {
		return nil, errors.New("could not find browser")
	}
	log.Printf("Found browser: %v", path)

	url, err := launcher.New().Bin(path).Launch()
	if err != nil {
		return nil, fmt.Errorf("could not launch browser: %w", err)
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("could not connect to browser: %w", err)
	}

	return browser, nil
}

func readConfig() (config, error) {
	hassBaseURL := os.Getenv("HS_HASS_BASE_URL")
	if hassBaseURL == "" {
		return config{}, errors.New("missing HASS base URL")
	}

	hassToken := os.Getenv("HS_HASS_TOKEN")
	if hassToken == "" {
		return config{}, errors.New("missing HASS token")
	}

	hassPaths := strings.Split(os.Getenv("HS_HASS_PATHS"), ",")

	var scales []float64
	scalesEnv := strings.Split(os.Getenv("HS_SCALES"), ",")
	for _, s := range scalesEnv {
		var scale float64
		if s == "" {
			scale = 1
		} else {
			var err error
			scale, err = strconv.ParseFloat(s, 64)
			if err != nil {
				return config{}, fmt.Errorf("invalid scale: %w", err)
			}
		}
		scales = append(scales, scale)
	}
	if len(scales) != 1 && len(scales) != len(hassPaths) {
		return config{}, fmt.Errorf("scales (%v) do not match paths (%v)", len(scales), len(hassPaths))
	}

	var width int
	widthEnv := os.Getenv("HS_WIDTH")
	if widthEnv == "" {
		width = 480
	} else {
		var err error
		width, err = strconv.Atoi(widthEnv)
		if err != nil {
			return config{}, fmt.Errorf("invalid width: %w", err)
		}
	}

	var height int
	heightEnv := os.Getenv("HS_HEIGHT")
	if heightEnv == "" {
		height = 800
	} else {
		var err error
		height, err = strconv.Atoi(heightEnv)
		if err != nil {
			return config{}, fmt.Errorf("invalid height: %w", err)
		}
	}

	listenAddr := os.Getenv("HS_LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8000"
	}

	cfg := config{
		hassBaseURL: hassBaseURL,
		hassToken:   hassToken,
		hassPaths:   hassPaths,
		scales:      scales,
		width:       width,
		height:      height,
		listenAddr:  listenAddr,
	}

	return cfg, nil
}
