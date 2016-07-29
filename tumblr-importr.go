package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/kezhuw/toml"
)

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

	die(json.Unmarshal(post, &data), "bad data from Tumblr")
	var m map[string]interface{}
	die(json.Unmarshal(post, &m), "bad data from Tumblr")

	date, err := time.Parse("2006-01-02 15:04:05 GMT", data.Date)
	die(err, "bad data from Tumblr")

	u, err := url.Parse(data.Post_url)
	die(err, "bad data from Tumblr")

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
	die(err, "could not save file")
	defer f.Close()

	fmt.Fprintln(f, "+++")
	t := toml.NewEncoder(f)
	die(t.Encode(output), "could not save file")
	fmt.Fprintln(f, "+++")
}
