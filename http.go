package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

// Client for all HTTP requests
var cl = &http.Client{
	// 5 second timeout is pretty generous
	Timeout: 5 * time.Second,
}

func get(url string, w io.Writer) (err error) {
	log.Printf("GET %s", url)

	rsp, err := cl.Get(url)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("could not fetch %s", url))
		return
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status for %s: %s", url, rsp.Status)
		return
	}

	if _, err = io.Copy(w, rsp.Body); err != nil {
		err = errors.Wrap(err, fmt.Sprintf("connection reset for %s", url))
		return
	}

	return
}

// Concurrency limiter for API fetcher
var (
	// Limit to 10 concurrent requests
	semaphore = make(chan bool, 10)
)

func fetch(url string) (io.Reader, error) {
	semaphore <- true
	defer func() {
		<-semaphore
	}()

	var buf bytes.Buffer
	err := get(url, &buf)

	return &buf, err
}

// Concurrency limiter for CDN file saver
var (
	// Self-limit to one request every 50ms
	rate     = 50 * time.Millisecond
	throttle = time.Tick(rate) // Leaks a routine (shrug)
)

func save(url, fullFilePath string) (err error) {
	// First try to make the directory
	dirname := filepath.Dir(fullFilePath)

	if err = os.MkdirAll(dirname, os.ModePerm); err != nil {
		return err
	}

	// Wait for throttling
	<-throttle

	log.Printf("Saving %s", fullFilePath)

	// Open file
	f, err := os.Create(fullFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Actually save the file
	err = get(url, f)

	return
}
