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

	// Make a post directory for saving things
	die(errors.Wrap(
		os.MkdirAll("post", os.ModePerm),
		"could not make posts directory"))

	tc := NewTumblrClient(blog, key)

	pc, ec := tc.Posts()

	for {
		select {
		case err = <-ec:
			die(err)
		case posts, ok := <-pc:
			if !ok {
				fmt.Println()
				return
			}
			fmt.Print(".")
			for _, post := range posts {
				die(processPost(post))
			}
		}
	}

	// TODO:
	// Split into three files: main, tumblr, savr
	// Save images somewhere
	// - Regex to find them in the TOML
	// - MD5 hash the image URLs to make a new file name
	// - Download and save them someplace
}

func die(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v", err)
		os.Exit(1)
	}
}
