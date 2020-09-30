package tumblr

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/carlmjohnson/errutil"
)

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
		go app.getPosts(ctx, offsetsCh, tumblrRespCh)
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

func (app *appEnv) getPosts(ctx context.Context, offsetsCh <-chan int, respCh chan<- tumblrResp) {
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
