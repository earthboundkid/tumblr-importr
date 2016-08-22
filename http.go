package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

var (
	// Limit to 10 concurrent requests
	semaphore = make(chan bool, 10)

	ctx, cancel = context.WithCancel(context.Background())
	// 15 second timeout is pretty generous
	cl = &http.Client{
		Timeout: 15 * time.Second,
	}
)

func fetch(url string, w io.Writer) (err error) {
	log.Printf("GET %s", url)

	semaphore <- true
	defer func() {
		<-semaphore
	}()

	req, _ := http.NewRequest("GET", url, nil)
	req = req.WithContext(ctx)
	rsp, err := cl.Do(req)
	if err != nil {
		cancel()
		err = errors.Wrap(err, fmt.Sprintf("could not fetch %s", url))
		return
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		cancel()
		return fmt.Errorf("bad status for %s: %s", url, rsp.Status)
	}

	if _, err = io.Copy(w, rsp.Body); err != nil {
		cancel()
		return errors.Wrap(err, fmt.Sprintf("connection reset for %s", url))
	}

	return
}
