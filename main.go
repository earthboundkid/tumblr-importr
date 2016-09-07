package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kezhuw/toml"
	"github.com/pkg/errors"
)

func main() {
	data, err := ioutil.ReadFile("./tumblr.toml")
	die(errors.Wrap(err, "could not read tumblr.toml"))

	var (
		blog, key string
	)

	err = toml.Unmarshal(data, &struct {
		Blog *string
		Key  *string
	}{&blog, &key})
	die(errors.Wrap(err, "could not parse config file"))

	// Todo configure me
	// TODO what if you don't want images?
	i := NewImageProcessor()
	pp := NewPostProcessor(i)
	tc := NewTumblrClient(blog, key, pp)

	die(tc.Wait())

	// TODO:
	// Move into sub-directory
	// Write documentation

	// Benchmark networking code
}

func die(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}
