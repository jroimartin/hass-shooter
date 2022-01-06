package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// HassShooter represents a Home Assistant screenshot capture web server.
type HassShooter struct {
	// cfg contains the HassShooter configuration.
	cfg Config

	// browser is the internal headless browser.
	browser *rod.Browser

	// cache is the internal image cache.
	cache *ImageCache
}

// NewHassShooter returns a new HassShooter with the provided configuration.
func NewHassShooter(cfg Config) (*HassShooter, error) {
	path, ok := launcher.LookPath()
	if !ok {
		return nil, errors.New("could not find browser")
	}

	url, err := launcher.New().Bin(path).Launch()
	if err != nil {
		return nil, fmt.Errorf("could not launch browser: %w", err)
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("could not connect to browser: %w", err)
	}

	if err := browser.IgnoreCertErrors(cfg.IgnoreCertErrors); err != nil {
		return nil, fmt.Errorf("could not set cert checking: %w", err)
	}

	hs := &HassShooter{
		cfg:     cfg,
		browser: browser,
		cache:   NewImageCache(len(cfg.HassPages)),
	}
	return hs, nil
}

// Close closes the internal headless browser.
func (hs *HassShooter) Close() error {
	if err := hs.browser.Close(); err != nil {
		return fmt.Errorf("coult not close browser: %w", err)
	}
	return nil
}

// ListenAndServe starts the HassShooter web server. First it will try to log
// into Home Assistant, then it will start a goroutine that refreshes the
// internal image cache periodically.
func (hs *HassShooter) ListenAndServe() error {
	logf("Logging into Home Assistant")
	if err := hs.hassLogin(); err != nil {
		return fmt.Errorf("could not log into hass: %w", err)
	}
	go func() {
		for {
			hs.refreshAll()
			time.Sleep(time.Duration(hs.cfg.RefreshTimeSecs) * time.Second)
		}
	}()

	http.Handle("/", hs.cache)
	return http.ListenAndServe(hs.cfg.ListenAddr, nil)
}

// refreshAll refreshes the internal image cache.
func (hs *HassShooter) refreshAll() {
	var wg sync.WaitGroup
	for i, page := range hs.cfg.HassPages {
		wg.Add(1)
		go func(i int, page Page) {
			defer wg.Done()
			hs.refresh(i, page)
		}(i, page)
	}
	wg.Wait()
}

// refresh refreshes the image of the internal image cache at index idx.
func (hs *HassShooter) refresh(idx int, page Page) {
	logf("Taking screenshot (%v)", page.Path)
	img, err := hs.screenshot(page)
	if err != nil {
		logf("could not take screenshot: %v", err)
		return
	}

	logf("Transforming image (%v)", page.Path)
	img, err = hs.transform(img)
	if err != nil {
		logf("could not transform image: %v", err)
		return
	}

	logf("Updating cache (%v)", page.Path)
	if err := hs.cache.Set(idx, img); err != nil {
		logf("could not cache image: %v", err)
		return
	}
}

// screenshot takes a screnshot of the specified page.
func (hs *HassShooter) screenshot(page Page) ([]byte, error) {
	timeout := time.Duration(hs.cfg.TimeoutSecs) * time.Second
	browser := hs.browser.Timeout(timeout)

	url := hs.cfg.HassBaseURL + page.Path
	bpage, err := browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return nil, fmt.Errorf("could not open page: %w", err)
	}
	defer bpage.Close()

	scale := page.Scale
	if scale == 0 {
		scale = 1
	}

	metrics := &proto.EmulationSetDeviceMetricsOverride{
		Width:             int(float64(hs.cfg.Width) / scale),
		Height:            int(float64(hs.cfg.Height) / scale),
		DeviceScaleFactor: scale,
	}
	if err := bpage.SetViewport(metrics); err != nil {
		return nil, fmt.Errorf("could not set window size: %w", err)
	}

	idletime := time.Duration(hs.cfg.MinIdleTimeSecs) * time.Second
	waitFcn := bpage.WaitRequestIdle(idletime, nil, nil)
	waitFcn()

	img, err := bpage.Screenshot(false, &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
	})
	if err != nil {
		return nil, fmt.Errorf("could not take screenshot: %w", err)
	}

	return img, nil
}

// transform transforms the provided PNG into a BMP suitable for e-ink
// displays.
func (hs *HassShooter) transform(png []byte) ([]byte, error) {
	var buf bytes.Buffer
	cmd := exec.Command(
		"convert",
		"png:-",
		"-resize", fmt.Sprintf("%dx%d!", hs.cfg.Width, hs.cfg.Height),
		"-rotate", fmt.Sprintf("%d", hs.cfg.Rotation),
		"-monochrome",
		"bmp:-",
	)
	cmd.Stdin = bytes.NewReader(png)
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("imagemagick error: %w", err)
	}
	return buf.Bytes(), nil
}

// hassLogin logs into Home Assistant.
func (hs *HassShooter) hassLogin() error {
	timeout := time.Duration(hs.cfg.TimeoutSecs) * time.Second
	browser := hs.browser.Timeout(timeout)

	target := proto.TargetCreateTarget{URL: hs.cfg.HassBaseURL}
	bpage, err := browser.Page(target)
	if err != nil {
		return fmt.Errorf("could not open page: %w", err)
	}
	defer bpage.Close()

	hassTokens := struct {
		HASSURL     string `json:"hassUrl"`
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}{
		HASSURL:     hs.cfg.HassBaseURL,
		AccessToken: hs.cfg.HassToken,
		TokenType:   "Bearer",
	}
	hassTokensData, err := json.Marshal(hassTokens)
	if err != nil {
		return fmt.Errorf("could not marshal hassTokens: %w", err)
	}

	script := `localStorage.setItem("hassTokens", arguments[0]);`
	args := []interface{}{string(hassTokensData)}
	if _, err := bpage.Eval(script, args); err != nil {
		return fmt.Errorf("could not execute script: %w", err)
	}

	return nil
}
