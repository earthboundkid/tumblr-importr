package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/pkg/errors"

	"golang.org/x/sync/errgroup"
)

const (
	apiURL   = `https://api.tumblr.com/v2/blog/%s/posts?api_key=%s&apiLimit=%d`
	apiLimit = 20
)

type TumblrClient struct {
	baseUrl *url.URL
}

func NewTumblrClient(blog, key string) TumblrClient {

	// Figure out starting URL
	baseUrl, _ := url.Parse(fmt.Sprintf(apiURL, blog, key, apiLimit))

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
	for i := 0 + apiLimit; i < totalPosts; i += apiLimit {
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
