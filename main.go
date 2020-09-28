package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/carlmjohnson/tumblr-importr/http"
	"github.com/carlmjohnson/tumblr-importr/tumblr"
	"github.com/kezhuw/toml"
	"github.com/pkg/errors"
)

const debug = true

func run() error {
	if !debug {
		log.SetOutput(ioutil.Discard)
	}

	log.Println("Reading toml...")

	data, err := ioutil.ReadFile("./tumblr.toml")
	if err != nil {
		return errors.Wrap(err, "could not read tumblr.toml")
	}

	var (
		blog, key, imageBaseURL, localImagePath string
	)

	err = toml.Unmarshal(data, &struct {
		Blog *string
		Key  *string
		ImageBaseURL *string
		LocalImagePath *string
	}{&blog, &key, &imageBaseURL, &localImagePath})
	if err != nil {
		return errors.Wrap(err, "could not parse config file")
	}

	log.Println("Starting processors")

	// Todo configure me
	// TODO what if you don't want images?
	i := tumblr.NewImageProcessor(imageBaseURL, localImagePath)
	pp := tumblr.NewPostProcessor(i)
	tc := tumblr.NewTumblrClient(blog, key, pp)

	return Countdown(http.BytesDownloaded, tc.Wait)

	// TODO:
	// Configuration options
	// Templates for output dir
	// Taskpools
	// Write documentation
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}

func Countdown(bytesDownloaded *int64, f func() error) error {
	ec := make(chan error)
	go func() {
		ec <- f()
	}()

	// Move cursor down one before starting
	fmt.Println()
	const (
		cursorUp  = "\033[1A"
		eraseLine = "\033[K"
	)

	ticker := time.NewTicker(100 * time.Millisecond)
	start := time.Now()
	var lastBdl, lastSpeed Size

	for {
		select {
		case <-ticker.C:
			bdl := Size(atomic.LoadInt64(bytesDownloaded))
			// Average speed with last speed and total speed
			totalSpeed := Size(bdl) / Size(time.Since(start)) * Size(time.Second)
			speed := Size(bdl-lastBdl) * 10
			speed = (lastSpeed + totalSpeed + speed) / 3
			fmt.Printf("%s\rDownloaded %s @ %s/s%s\n",
				cursorUp, bdl, speed, eraseLine)
			lastBdl, lastSpeed = bdl, speed
		case err := <-ec:
			return err
		}
	}
}
