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
	pp      *PostProcessor
	eg      errgroup.Group
}

func NewTumblrClient(blog, key string, pp *PostProcessor) *TumblrClient {
	// Figure out starting URL
	baseUrl, _ := url.Parse(fmt.Sprintf(apiURL, blog, key, apiLimit))

	return &TumblrClient{baseUrl: baseUrl, pp: pp}
}

func (tc *TumblrClient) getTumblrPosts(offset int) (posts []Post, totalPosts int, err error) {
	u := *tc.baseUrl
	q := u.Query()
	q.Set("offset", strconv.Itoa(offset))
	u.RawQuery = q.Encode()

	r, err := fetch(u.String())
	if err != nil {
		return
	}

	dec := json.NewDecoder(r)
	var s struct {
		Response struct {
			Total_posts int
			Posts       []Post
		}
	}

	err = errors.Wrap(dec.Decode(&s), "could not decode "+u.String())

	return s.Response.Posts, s.Response.Total_posts, err
}

func (tc *TumblrClient) Wait() error {
	// Fetch first page
	posts, totalPosts, err := tc.getTumblrPosts(0)
	if err != nil {
		tc.pp.Error(err)
		return err
	}

	// Send posts off to be processed
	tc.pp.Posts(posts)

	// Tell it to fetch other pages
	for i := 0 + apiLimit; i < totalPosts; i += apiLimit {
		offset := i
		tc.eg.Go(func() error {
			posts, _, err := tc.getTumblrPosts(offset)
			if err != nil {
				tc.pp.Error(err)
				return err
			}

			tc.pp.Posts(posts)
			return nil
		})
	}

	if err := tc.eg.Wait(); err != nil {
		tc.pp.Error(err)
		return err
	}

	return tc.pp.Wait()
}
