package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/kezhuw/toml"
	"github.com/pkg/errors"
)

type Post struct {
	json.RawMessage
}

func processPost(post Post) (err error) {
	var data struct {
		Date     string
		Id       int
		Post_url string
		Slug     string
		Tags     []string
		Title    string
		Type     string
	}

	if err = json.Unmarshal(post.RawMessage, &data); err != nil {
		err = errors.Wrap(err, "bad data from Tumblr")
		return
	}

	var m map[string]interface{}
	if err = json.Unmarshal(post.RawMessage, &m); err != nil {
		err = errors.Wrap(err, "bad data from Tumblr")
		return
	}

	date, err := time.Parse("2006-01-02 15:04:05 GMT", data.Date)
	if err != nil {
		err = errors.Wrap(err, "bad data from Tumblr")
		return
	}

	u, err := url.Parse(data.Post_url)
	if err != nil {
		err = errors.Wrap(err, "bad data from Tumblr")
		return
	}

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

	path := fmt.Sprintf("post/%4d/%02d/", date.Year(), date.Month())
	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		err = errors.Wrap(err, "could not make directory to save entries in")
		return
	}

	fname := fmt.Sprintf("%s/%d-%s.md", path, data.Id, data.Slug)
	f, err := os.Create(fname)
	if err != nil {
		err = errors.Wrap(err, "could not save file")
		return
	}

	defer f.Close()

	fmt.Fprintln(f, "+++")
	t := toml.NewEncoder(f)
	if err = t.Encode(output); err != nil {
		err = errors.Wrap(err, "could not save file")
		return
	}

	fmt.Fprintln(f, "+++")
	return
}
