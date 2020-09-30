package tumblr

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/carlmjohnson/flagext"
)

func CLI(args []string) error {
	var app appEnv
	err := app.ParseArgs(args)
	if err != nil {
		return err
	}
	if err = app.Exec(); err != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %+v\n", err)
	}
	return err
}

type appEnv struct {
	client            *Client
	localPostPath     string
	imageBaseURL      string
	imageRewritePath  string
	localImagePath    string
	workers           int
	skipAPIRequest    bool
	skipImageDownload bool
}

const AppName = "tumblr-importr"

func (app *appEnv) ParseArgs(args []string) error {
	fl := flag.NewFlagSet(AppName, flag.ContinueOnError)
	fl.Usage = func() {
		fmt.Fprintf(fl.Output(), `tumblr-importr

A tool for creating a Hugo blog from a Tumblr site.

Options can also be set as environmental variables named like
TUMBLR_IMPORTR_API_KEY. An API key must be created by registering at
<http://www.tumblr.com/oauth/apps>.

Downloading images can take a long time, so separate options exist for skipping
downloading posts or images. Image downloads will resume if restarted.

Options:

`)
		fl.PrintDefaults()
		fmt.Fprintln(fl.Output(), "")
	}

	blog := fl.String("blog", "", "`blog name` to import")
	key := fl.String("api-key", "", "Tumblr consumer API `key`")
	fl.StringVar(&app.localPostPath, "post-dest", "content/post", "destination `path` to save posts")
	fl.StringVar(&app.localImagePath, "image-dest", "static/images", "destination `path` to save images")
	fl.StringVar(&app.imageBaseURL, "image-url", "/images", "new base `URL` for images")
	fl.StringVar(&app.imageRewritePath, "image-rewrites", "image-rewrites.json", "`path` for JSON file containing image rewrites")
	fl.DurationVar(&http.DefaultClient.Timeout, "timeout", 10*time.Second, "HTTP client timeout")
	trans := http.DefaultTransport.(*http.Transport)
	fl.IntVar(&trans.MaxConnsPerHost, "max-conns-per-host", runtime.NumCPU()+1, "max number of simultaneous connections")
	fl.IntVar(&app.workers, "workers", runtime.NumCPU()+1, "number of workers")
	fl.BoolVar(&app.skipAPIRequest, "skip-api-request", false,
		"skip downloading posts from the API and just use -image-rewrites file to download images")
	fl.BoolVar(&app.skipImageDownload, "skip-image-download", false,
		"skip downloading images (still rewrites)")

	if err := fl.Parse(args); err != nil {
		return err
	}
	if err := flagext.ParseEnv(fl, AppName); err != nil {
		return err
	}
	if err := flagext.MustHave(fl, "api-key", "blog"); err != nil {
		return err
	}
	app.client = NewClient(*blog, *key, http.DefaultClient)
	return nil
}

func (app *appEnv) log(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func (app *appEnv) Exec() error {
	if app.skipAPIRequest && app.skipImageDownload {
		app.log("nothing to do\n")
		return nil
	}
	if app.skipAPIRequest {
		return app.loadImageSubs()
	}
	return app.apiRequest()
}
