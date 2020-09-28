package tumblr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/carlmjohnson/tumblr-importr/httpjson"
)

type Client struct {
	baseURL *url.URL
	cl      *http.Client
}

const (
	apiURL   = `https://api.tumblr.com/v2/blog/%s/posts?api_key=%s&apiLimit=%d`
	apiLimit = 20
)

func NewClient(blog, key string, cl *http.Client) *Client {
	// Figure out starting URL
	baseURL, _ := url.Parse(fmt.Sprintf(apiURL, blog, key, apiLimit))
	if cl == nil {
		cl = http.DefaultClient
	}
	return &Client{baseURL, cl}
}

func (tc *Client) GetOffset(ctx context.Context, offset int) (resp APIResponse, err error) {
	u := *tc.baseURL
	q := u.Query()
	q.Set("offset", strconv.Itoa(offset))
	u.RawQuery = q.Encode()

	var data APIEnvelope
	err = httpjson.Get(ctx, tc.cl, u.String(), &data)

	return data.Response, err
}

type APIEnvelope struct {
	Response APIResponse `json:"response"`
}

type APIResponse struct {
	Total int    `json:"total_posts"`
	Posts []Post `json:"posts"`
}

type Post struct {
	json.RawMessage
}
