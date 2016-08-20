package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

func fetch(url string) (io.Reader, error) {
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
		return nil, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		cancel()
		return nil, fmt.Errorf("bad status for %s: %s", url, rsp.Status)
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, rsp.Body); err != nil {
		cancel()
		return nil, errors.Wrap(err, fmt.Sprintf("connection reset for %s", url))
	}

	return &buf, nil
}
