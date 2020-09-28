package tumblr

import (
	"crypto/md5"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/carlmjohnson/tumblr-importr/http"
	"github.com/pkg/errors"

	"golang.org/x/sync/errgroup"
)

type imageProcessor struct {
	baseURL   string
	imagePath string
	seenURLs  map[string]bool
	m         sync.Mutex
	eg        errgroup.Group
}

func NewImageProcessor(imageBaseURL string, localImagePath string) *imageProcessor {
	return &imageProcessor{
		baseURL:   imageBaseURL,
		imagePath: localImagePath,
		seenURLs:  map[string]bool{},
	}
}

func (ip *imageProcessor) Replace(originalURL []byte) []byte {
	originalURLstr := string(originalURL)
	hashedFileName := fmt.Sprintf("%x%s",
		md5.Sum(originalURL), filepath.Ext(originalURLstr))

	fullFilePath := filepath.Join(ip.imagePath, hashedFileName[0:2],
		hashedFileName[2:4], hashedFileName[4:])

	ip.add(originalURLstr, fullFilePath)

	newURL := fmt.Sprintf("%s%s/%s/%s", ip.baseURL, hashedFileName[0:2],
		hashedFileName[2:4], hashedFileName[4:])

	return []byte(newURL)
}

func (ip *imageProcessor) add(originalURLstr, fullFilePath string) {
	ip.m.Lock()
	defer ip.m.Unlock()

	if ip.seenURLs[originalURLstr] {
		return
	}

	ip.seenURLs[originalURLstr] = true

	ip.eg.Go(func() (err error) {
		for i := 0; i < 3; i++ {
			err = http.Save(originalURLstr, fullFilePath)
			if err == nil {
				break
			}
			// Hmm, something went wrong, try again after sleeping
			time.Sleep(500 * time.Millisecond)
		}

		return errors.Wrap(err, "Repeatedly failed to fetch")
	})
}

func (ip *imageProcessor) Wait() error {
	return ip.eg.Wait()
}
