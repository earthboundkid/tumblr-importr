package tumblr

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/carlmjohnson/errutil"
	"github.com/henvic/ctxsignal"
)

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
	ctx, cancel := ctxsignal.WithTermination(context.Background())
	defer cancel()

	var (
		errors           errutil.Slice
		total            = len(imgSubs)
		inflightRequests = 0
		skippedN         = 0
		subCh            = make(chan imgSub)
		errCh            = make(chan error)
	)
	for i := 0; i < app.workers; i++ {
		go app.getImage(ctx, subCh, errCh)
	}
loop:
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

		case <-ctx.Done():
			// drain workers
			for i := 0; i < inflightRequests; i++ {
				err := <-errCh
				if err == errSkip {
					err = nil
					skippedN++
				}
				errors.Push(err)
			}

			break loop
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

func (app *appEnv) getImage(ctx context.Context, subCh chan imgSub, errCh chan error) {
	for sub := range subCh {
		err := save(ctx, http.DefaultClient, sub.url, sub.destPath)
		errCh <- err
	}
}
