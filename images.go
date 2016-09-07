package main

import (
	"crypto/md5"
	"fmt"
	"path/filepath"
	"sync"

	"golang.org/x/sync/errgroup"
)

type imageProcessor struct {
	baseURL   string
	imagePath string
	seenURLs  map[string]bool
	m         sync.Mutex
	eg        errgroup.Group
}

func NewImageProcessor() *imageProcessor {
	return &imageProcessor{
		baseURL:   "https://example.com/images/",
		imagePath: "images/",
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
		return save(originalURLstr, fullFilePath)
		// 	for i := 0; i < 3; i++ {
		// 		err = save(originalURLstr, fullFilePath)
		// 		if err == nil {
		// 			break
		// 		}
		// 		log.Printf("Fetch err: %v", err)
		// 		// Hmm, something went wrong, try again after sleeping
		// 		time.Sleep(500 * time.Millisecond)
		// 	}

		// 	return err
	})
}

func (ip *imageProcessor) Wait() error {
	return ip.eg.Wait()
}
