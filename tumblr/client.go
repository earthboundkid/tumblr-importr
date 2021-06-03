package tumblr

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/carlmjohnson/requests"
)

type Client struct {
	rb *requests.Builder
}

const (
	apiLimit = 20
)

func NewClient(blog, key string, cl *http.Client) *Client {
	return &Client{
		requests.
			URL("https://api.tumblr.com").
			Client(cl).
			Pathf("/v2/blog/%s/posts", blog).
			Param("api_key", key).
			Param("apiLimit", strconv.Itoa(apiLimit)),
	}
}

func (tc *Client) GetOffset(ctx context.Context, offset int) (resp APIResponse, err error) {
	var data APIEnvelope
	err = tc.rb.Clone().
		Param("offset", strconv.Itoa(offset)).
		ToJSON(&data).
		Fetch(ctx)

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
