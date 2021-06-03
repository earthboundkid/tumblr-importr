package tumblr

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/carlmjohnson/errutil"
	"github.com/carlmjohnson/requests"
)

var errSkip = errors.New("skip")

func save(ctx context.Context, cl *http.Client, url, fullFilePath string) (err error) {
	// Skip if it exists
	if info, err := os.Stat(fullFilePath); err == nil && info.Size() != 0 {
		return errSkip
	}
	// First try to make the directory
	dirname := filepath.Dir(fullFilePath)
	if err = os.MkdirAll(dirname, os.ModePerm); err != nil {
		return
	}
	// Open file
	f, err := os.Create(fullFilePath)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.Remove(fullFilePath)
		}
	}()
	defer errutil.Defer(&err, f.Close)

	return requests.
		URL(url).
		ToWriter(f).
		Fetch(ctx)
}
