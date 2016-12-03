package tumblr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/kezhuw/toml"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type Post struct {
	json.RawMessage
}

type PostProcessor struct {
	ip *imageProcessor
	eg errgroup.Group
}

func NewPostProcessor(ip *imageProcessor) *PostProcessor {
	return &PostProcessor{ip: ip}
}

func (pp *PostProcessor) Posts(posts []Post) {
	for _, post := range posts {
		post := post
		pp.eg.Go(func() error {
			return pp.processPost(post)
		})
	}
}

func (pp *PostProcessor) Error(err error) {
	if err != nil {
		pp.eg.Go(func() error {
			return err
		})
	}
}

func (pp *PostProcessor) Wait() error {
	if err := pp.eg.Wait(); err != nil {
		return err
	}
	return pp.ip.Wait()
}

var reg = regexp.MustCompile(`https?://[\w-.]+tumblr\.com/[\w-./]+(\.jpe?g|\.png|\.gif)`)

func (pp *PostProcessor) processPost(post Post) (err error) {
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

	b, err := toml.Marshal(output)
	if err != nil {
		return errors.Wrap(err, "TOML error")
	}

	b = reg.ReplaceAllFunc(b, pp.ip.Replace)
	r := bytes.NewReader(b)

	// Todo: use a template
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

	if _, err = io.WriteString(f, "+++\n"); err != nil {
		return
	}
	if _, err = io.Copy(f, r); err != nil {
		return
	}
	if _, err = io.WriteString(f, "+++\n"); err != nil {
		return
	}

	return

}
