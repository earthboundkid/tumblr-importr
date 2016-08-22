package main

import (
	"crypto/md5"
	"fmt"
	"os"
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
	i := imageProcessor{
		baseURL:   "https://example.com/images/",
		imagePath: "images/",
		seenURLs:  map[string]bool{},
	}
	// TODO
	_ = os.MkdirAll(i.imagePath, os.ModePerm)
	return &i
}

func (i *imageProcessor) Replace(originalURL []byte) []byte {
	originalURLstr := string(originalURL)
	hashedFileName := fmt.Sprintf("%x%s",
		md5.Sum(originalURL), filepath.Ext(originalURLstr))

	i.add(originalURLstr, hashedFileName)

	newURL := i.baseURL + hashedFileName

	return []byte(newURL)
}

func (i *imageProcessor) add(originalURLstr, hashedFileName string) {
	i.m.Lock()
	defer i.m.Unlock()

	if i.seenURLs[originalURLstr] {
		return
	}

	i.seenURLs[originalURLstr] = true

	i.eg.Go(func() error {
		f, err := os.Create(i.imagePath + hashedFileName)
		if err != nil {
			return err
		}

		return fetch(originalURLstr, f)
	})
}

func (i *imageProcessor) Wait() error {
	return i.eg.Wait()
}
