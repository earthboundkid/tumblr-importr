package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"sync"

	"github.com/kezhuw/toml"
)

func main() {
	const apiURL = `https://api.tumblr.com/v2/blog/%s/posts?api_key=%s&limit=%d`

	data, err := ioutil.ReadFile("./config.toml")
	die(err, "could not read config.toml")

	var (
		blog, key string
		limit     = 20
	)

	err = toml.Unmarshal(data, &struct {
		Blog *string
		Key  *string
	}{&blog, &key})
	die(err, "could not parse config file")

	// Make a post directory for saving things
	die(os.MkdirAll("post", os.ModePerm), "could not make posts directory")

	// Figure out starting URL
	u, err := url.Parse(fmt.Sprintf(apiURL, blog, key, limit))
	die(err, "bad blog name or API key")
	baseUrl = u

	// Fetch first page
	posts, totalPosts, err := getTumblrPosts(0)
	die(err, "connection error")

	var (
		pc = make(chan []json.RawMessage, 1)
		wg sync.WaitGroup
	)

	// Send posts off to be processed
	pc <- posts

	// Tell it to fetch other pages
	for i := 0 + limit; i < totalPosts; i += limit {
		wg.Add(1)
		go func(offset int) {
			posts, _, err := getTumblrPosts(offset)
			die(err, "connection error")
			pc <- posts
			wg.Done()
		}(i)
	}

	go func() {
		wg.Wait()
		close(pc)
	}()

	for posts := range pc {
		for _, post := range posts {
			processPost(post)
		}
	}

	// TODO:
	// Split into three files: main, tumblr, savr
	// Save images somewhere
	// - Regex to find them in the TOML
	// - MD5 hash the image URLs to make a new file name
	// - Download and save them someplace
}

func die(err error, msg string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s: %v", msg, err)
		os.Exit(1)
	}
}
