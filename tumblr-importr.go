package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/kezhuw/toml"
)

func die(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func processPost(post json.RawMessage) {
	var data struct {
		Date     string
		Id       int
		Post_url string
		Slug     string
		Tags     []string
		Title    string
		Type     string
	}

	die(json.Unmarshal(post, &data))
	var m map[string]interface{}
	die(json.Unmarshal(post, &m))

	date, err := time.Parse("2006-01-02 15:04:05 GMT", data.Date)
	die(err)

	u, err := url.Parse(data.Post_url)
	die(err)

	var output = struct {
		Date    time.Time   `toml:"date"`
		Title   string      `toml:"title,omitempty"`
		Slug    string      `toml:"slug,omitempty"`
		Id      int         `toml:"id,string"`
		Aliases []string    `toml:"aliases"`
		Tags    []string    `toml:"tags"`
		Type    string      `toml:"type"`
		Tumblr  interface{} `toml:"tumblr,multiline"`
	}{
		date,
		data.Title,
		data.Slug,
		data.Id,
		[]string{u.Path},
		data.Tags,
		"tumblr-" + data.Type,
		m,
	}

	fname := fmt.Sprintf("post/%d.md", data.Id)
	f, err := os.Create(fname)
	die(err)
	defer f.Close()

	fmt.Fprintln(f, "+++")
	t := toml.NewEncoder(f)
	die(t.Encode(output))
	fmt.Fprintln(f, "+++")
}

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

	err = dec.Decode(&s)
	if err != nil {
		log.Fatal(u.String())
		return
	}

	return s.Response.Posts, s.Response.Total_posts, nil
}

func main() {
	const apiURL = `https://api.tumblr.com/v2/blog/%s/posts?api_key=%s&limit=%d`

	data, err := ioutil.ReadFile("./config.toml")
	die(err)

	var (
		blog, key string
		limit     = 20
	)

	err = toml.Unmarshal(data, &struct {
		Blog *string
		Key  *string
	}{&blog, &key})
	die(err)

	// Make a post directory for saving things
	die(os.MkdirAll("post", os.ModePerm))

	// Figure out starting URL
	u, err := url.Parse(fmt.Sprintf(apiURL, blog, key, limit))
	die(err)
	baseUrl = u

	// Fetch first page
	posts, totalPosts, err := getTumblrPosts(0)
	die(err)

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
			die(err)
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
