package main

import (
	"os"

	"github.com/carlmjohnson/exitcode"
	"github.com/carlmjohnson/tumblr-importr/tumblr"
)

func main() {
	exitcode.Exit(tumblr.CLI(os.Args[1:]))
}
