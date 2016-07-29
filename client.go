package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// Limit to 10 concurrent requests
var semaphore = make(chan bool, 10)

func fetch(url string) (io.Reader, error) {
	semaphore <- true
	defer func() {
		<-semaphore
	}()

	rsp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	var buf bytes.Buffer
	io.Copy(&buf, rsp.Body)

	return &buf, nil
}

var baseUrl *url.URL

func getTumblrPosts(offset int) (posts []json.RawMessage, totalPosts int, err error) {
	u := *baseUrl
	q := u.Query()
	q.Set("offset", strconv.Itoa(offset))
	u.RawQuery = q.Encode()

	rsp, err := fetch(u.String())
	if err != nil {
		return
	}

	dec := json.NewDecoder(rsp)
	var s struct {
		Response struct {
			Total_posts int
			Posts       []json.RawMessage
		}
	}

	die(dec.Decode(&s), "could not decode URL "+u.String())

	return s.Response.Posts, s.Response.Total_posts, nil
}
