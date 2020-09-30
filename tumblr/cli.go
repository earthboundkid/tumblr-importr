package tumblr

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/carlmjohnson/errutil"
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

func (app *appEnv) apiRequest() error {
	app.log("Making initial request\n")

	ctx := context.Background()
	resp, err := app.client.GetOffset(ctx, 0)
	if err != nil {
		return err
	}
	app.log("Blog has %d posts\n\n", resp.Total)
	var (
		errors           errutil.Slice
		offsets          []int
		requestsInflight        = 0
		postsProcessing         = 0
		postsProcessed          = 0
		postsTotal              = resp.Total
		postsQueue       []Post = resp.Posts
		offsetsCh               = make(chan int)
		tumblrRespCh            = make(chan tumblrResp)
		postsCh                 = make(chan Post)
		processCh               = make(chan processResp)
		imgSubs                 = make(map[string]string)
	)
	// queue offsets
	for i := 0 + apiLimit; i < postsTotal; i += apiLimit {
		offsets = append(offsets, i)
	}
	// start workers
	for i := 0; i < app.workers; i++ {
		go app.getPage(ctx, offsetsCh, tumblrRespCh)
		go app.processPosts(postsCh, processCh)
	}

loop:
	for {
		if postsTotal == postsProcessed {
			break
		}
		if requestsInflight == 0 &&
			postsProcessing == 0 &&
			len(postsQueue) == 0 {
			break
		}
		var offset int
		loopOffsetCh := offsetsCh
		if len(offsets) > 0 {
			offset = offsets[0]
		} else {
			loopOffsetCh = nil
		}
		var post Post
		loopPostsCh := postsCh
		if len(postsQueue) > 0 {
			post = postsQueue[0]
		} else {
			loopPostsCh = nil
		}
		select {
		// get stuff from API
		case loopOffsetCh <- offset:
			offsets = offsets[1:]
			requestsInflight++
		case resp := <-tumblrRespCh:
			errors.Push(resp.error)
			postsQueue = append(postsQueue, resp.Posts...)
			requestsInflight--

		// process responses
		case loopPostsCh <- post:
			postsQueue = postsQueue[1:]
			postsProcessing++

		case procResp := <-processCh:
			// todo better handling
			errors.Push(procResp.error)
			for k, v := range procResp.imgSubs {
				imgSubs[k] = v
			}
			postsProcessing--
			postsProcessed++
		// todo
		case <-ctx.Done():
			break loop
		}

		app.log("\rPosts saved: %d/%d Errors: %d      ",
			postsProcessed, postsTotal, len(errors),
		)
	}
	app.log("\n\n")
	close(offsetsCh)
	close(tumblrRespCh)
	close(postsCh)
	close(processCh)

	imgData, _ := json.MarshalIndent(imgSubs, "", "  ")
	// todo error handling
	if err = ioutil.WriteFile(app.imageRewritePath, imgData, os.ModePerm); err != nil {
		errors.Push(err)
		return errors.Merge()
	}
	app.log("Wrote image replacements file %q\n", app.imageRewritePath)
	if err = errors.Merge(); err != nil {
		return err
	}
	if app.skipImageDownload {
		app.log("Skipping image downloads\n")
		return nil
	}
	return app.getImages(makeImgSubs(imgSubs))
}

type tumblrResp struct {
	APIResponse
	error
}

func (app *appEnv) getPage(ctx context.Context, offsetsCh <-chan int, respCh chan<- tumblrResp) {
	for offset := range offsetsCh {
		var resp tumblrResp
		resp.APIResponse, resp.error = app.client.GetOffset(ctx, offset)
		respCh <- resp
	}
}

type processResp struct {
	imgSubs map[string]string
	error
}

func (app *appEnv) processPosts(postsCh <-chan Post, respCh chan<- processResp) {
	for post := range postsCh {
		var resp processResp
		resp.imgSubs, resp.error = app.processPost(post)
		respCh <- resp
	}
}

func (app *appEnv) loadImageSubs() error {
	app.log("loading %q\n", app.imageRewritePath)
	imgSubData, err := ioutil.ReadFile(app.imageRewritePath)
	if err != nil {
		return err
	}
	var imgSubs map[string]string
	if err = json.Unmarshal(imgSubData, &imgSubs); err != nil {
		return err
	}
	return app.getImages(makeImgSubs(imgSubs))
}

type imgSub struct {
	url, destPath string
}

func makeImgSubs(m map[string]string) []imgSub {
	imgSubs := make([]imgSub, 0, len(m))
	for url, dest := range m {
		imgSubs = append(imgSubs, imgSub{url, dest})
	}
	return imgSubs
}

func (app *appEnv) getImages(imgSubs []imgSub) error {
	app.log("Saving %d images...\n\n", len(imgSubs))
	var (
		errors           errutil.Slice
		total            = len(imgSubs)
		inflightRequests = 0
		skippedN         = 0
		subCh            = make(chan imgSub)
		errCh            = make(chan error)
	)
	for i := 0; i < app.workers; i++ {
		go app.getImage(subCh, errCh)
	}
	for len(imgSubs) > 0 || inflightRequests > 0 {
		var loopSub imgSub
		loopCh := subCh
		if len(imgSubs) > 0 {
			loopSub = imgSubs[0]
		} else {
			loopCh = nil
		}
		select {
		case loopCh <- loopSub:
			inflightRequests++
			imgSubs = imgSubs[1:]
		// todo retries???
		case err := <-errCh:
			if err == errSkip {
				err = nil
				skippedN++
			}
			errors.Push(err)
			inflightRequests--
		}
		saved := total - len(imgSubs) - inflightRequests
		app.log("\rImages saved: %d/%d Skipped: %d Errors: %d      ",
			saved, total, skippedN, len(errors),
		)
	}
	app.log("\n\n")
	close(subCh)
	close(errCh)
	return errors.Merge()
}

func (app *appEnv) getImage(subCh chan imgSub, errCh chan error) {
	for sub := range subCh {
		err := save(http.DefaultClient, sub.url, sub.destPath)
		errCh <- err
	}
}
