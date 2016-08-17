package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"golang.org/x/sync/errgroup"
)

var (
	// Limit to 10 concurrent requests
	semaphore = make(chan bool, 10)

	// 5 second timeout is pretty generous
	cl = &http.Client{
		Timeout: 5 * time.Second,
	}
)

func fetch(url string) (io.Reader, error) {
	semaphore <- true
	defer func() {
		<-semaphore
	}()

	rsp, err := cl.Get(url)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("could not fetch %s", url))
		return nil, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status for %s: %s", url, rsp.Status)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, rsp.Body)

	return &buf, errors.Wrap(err, fmt.Sprintf("connection reset for %s", url))
}

type TumblrClient struct {
	baseUrl *url.URL
}

const (
	apiURL = `https://api.tumblr.com/v2/blog/%s/posts?api_key=%s&limit=%d`
	limit  = 20
)

func NewTumblrClient(blog, key string) TumblrClient {

	// Figure out starting URL
	baseUrl, _ := url.Parse(fmt.Sprintf(apiURL, blog, key, limit))

	return TumblrClient{baseUrl}
}

func (tc TumblrClient) getTumblrPosts(offset int) (posts []Post, totalPosts int, err error) {
	u := *tc.baseUrl
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
			Posts       []Post
		}
	}

	err = errors.Wrap(dec.Decode(&s), "could not decode "+u.String())

	return s.Response.Posts, s.Response.Total_posts, err
}

func (tc TumblrClient) Posts() <-chan PostProcessor {
	var (
		// Buffer channel so we can send before we return
		pp = make(chan PostProcessor, 1)
		eg errgroup.Group
	)

	// Fetch first page
	posts, totalPosts, err := tc.getTumblrPosts(0)
	if err != nil {
		pp <- PostProcessor{err: err}
		return pp
	}

	// Send posts off to be processed
	pp <- PostProcessor{posts: posts}

	// Tell it to fetch other pages
	for i := 0 + limit; i < totalPosts; i += limit {
		offset := i
		eg.Go(func() error {
			posts, _, err := tc.getTumblrPosts(offset)
			if err != nil {
				return err
			}

			pp <- PostProcessor{posts: posts}
			return nil
		})
	}

	go func() {
		defer close(pp)
		if err := eg.Wait(); err != nil {
			pp <- PostProcessor{err: err}
		}
	}()

	return pp
}
