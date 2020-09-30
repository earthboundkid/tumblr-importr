package tumblr

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/carlmjohnson/errutil"
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
	defer errutil.Defer(&err, f.Close)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	rsp, err := cl.Do(req)
	if err != nil {
		return
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status for %s: %s", url, rsp.Status)
		return
	}
	buf := bufio.NewReader(rsp.Body)
	if _, err = io.Copy(f, buf); err != nil {
		os.Remove(fullFilePath)
	}
	return
}
