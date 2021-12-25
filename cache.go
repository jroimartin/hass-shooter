package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// An Image is just a slice of bytes containing the data of the image.
type Image []byte

// NewImageReader returns an io.Reader pointing to the image data.
func NewImageReader(img Image) io.Reader {
	return bytes.NewReader(img)
}

// ImageCache is an image cache. It is safe for concurrent use by multiple
// goroutines.
type ImageCache struct {
	// images contains the images data.
	images []Image

	// mu guards images.
	mutex sync.RWMutex
}

// NewImageCache returns a new ImageCache with the specified capacity.
func NewImageCache(capacity int) *ImageCache {
	cache := &ImageCache{
		images: make([]Image, capacity),
	}
	return cache
}

// Set creates an image at index idx copying the specified data.
func (cache *ImageCache) Set(idx int, data []byte) error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if idx < 0 || idx >= len(cache.images) {
		return errors.New("index is out of bounds")
	}

	img := make(Image, len(data))
	copy(img, data)

	cache.images[idx] = img

	return nil
}

// Get returns a copy of the image at index idx. If the image has not been
// initialized with data, an error is returned.
func (cache *ImageCache) Get(idx int) (Image, error) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	if idx < 0 || idx >= len(cache.images) {
		return nil, errors.New("index is out of bounds")
	}

	if cache.images[idx] == nil {
		return nil, errors.New("uninitialized image")
	}

	img := make(Image, len(cache.images[idx]))
	copy(img, cache.images[idx])

	return img, nil
}

// ServeHTTP implements an http.Handler that serves an image from the cache. It
// expects the URL path to have the format "/idx", where idx is an integer that
// specifies the index of the image in the cache.
func (cache *ImageCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logf("%v - %v - %v %v",
		r.RemoteAddr, r.UserAgent(), r.Method, r.URL)

	path := strings.TrimPrefix(r.URL.Path, "/")

	var (
		idx int
		err error
	)
	if path == "" {
		idx = 0
	} else {
		idx, err = strconv.Atoi(path)
		if err != nil {
			logf("Could not parse index (%v)", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	img, err := cache.Get(idx)
	if err != nil {
		logf("Could not get image: %v", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(img)))

	if _, err := io.Copy(w, NewImageReader(img)); err != nil {
		logf("Could not write image: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
