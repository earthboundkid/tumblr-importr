package tumblr

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"

	"github.com/carlmjohnson/errutil"
	"github.com/kezhuw/toml"
)

type FrontMatter struct {
	Date    time.Time   `toml:"date"`
	Title   string      `toml:"title,omitempty"`
	Slug    string      `toml:"slug,omitempty"`
	Id      int         `toml:"id,string"`
	Aliases []string    `toml:"aliases"`
	Tags    []string    `toml:"tags"`
	Type    string      `toml:"type"`
	Tumblr  interface{} `toml:"tumblr,multiline"`
}

var tumblrImagesRE = regexp.MustCompile(`https?://[\w-.]+tumblr\.com/[\w-./]+(\.jpe?g|\.png|\.gif)`)

func (app *appEnv) processPost(post Post) (imgSubs map[string]string, err error) {
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
		err = wrap(err, "bad data from Tumblr")
		return
	}

	var m map[string]interface{}
	if err = json.Unmarshal(post.RawMessage, &m); err != nil {
		err = wrap(err, "bad data from Tumblr")
		return
	}

	date, err := time.Parse("2006-01-02 15:04:05 GMT", data.Date)
	if err != nil {
		err = wrap(err, "bad data from Tumblr")
		return
	}

	u, err := url.Parse(data.Post_url)
	if err != nil {
		err = wrap(err, "bad data from Tumblr")
		return
	}

	var output = FrontMatter{
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
		err = wrap(err, "TOML error")
		return
	}

	imgSubs = make(map[string]string)
	b = tumblrImagesRE.ReplaceAllFunc(b, func(b []byte) []byte {
		originalURLstr := string(b)
		hashedFileName := fmt.Sprintf("%x%s",
			md5.Sum(b), filepath.Ext(originalURLstr))
		// path, not filepath because it's a URL
		hashedFileName = path.Join(
			hashedFileName[0:2], hashedFileName[2:4], hashedFileName[4:])
		// filepath for portability?
		fullFilePath := filepath.Join(app.localImagePath, hashedFileName)

		imgSubs[originalURLstr] = fullFilePath

		newURL := path.Join(app.imageBaseURL, hashedFileName)

		return []byte(newURL)
	})
	r := bytes.NewReader(b)

	path := fmt.Sprintf("%s/%4d/%02d/",
		app.localPostPath, date.Year(), date.Month())
	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		err = wrap(err, "could not make directory to save entries in")
		return
	}

	fname := fmt.Sprintf("%s/%d-%s.md", path, data.Id, data.Slug)
	f, err := os.Create(fname)
	if err != nil {
		err = wrap(err, "could not save file")
		return
	}

	defer errutil.Defer(&err, f.Close)

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
