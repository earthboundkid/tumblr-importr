# tumblr-importr

An importer that uses the Tumblr API to create a Hugo static site

## Installation
First install [Go](http://golang.org).

If you just want to install the binary to your current directory and don't care about the source code, run

```bash
GOBIN="$(pwd)" GOPATH="$(mktemp -d)" go get github.com/carlmjohnson/tumblr-importr
```

This will create an executable called `tumblr-importr` in your current directory. Put it wherever you put executables.

## Usage
First, get a [Tumblr API key](http://www.tumblr.com/oauth/apps).

```bash
$ tumblr-importr --help
tumblr-importr

A tool for creating a Hugo blog from a Tumblr site.

Downloading images can take a long time, so separate options exist for skipping
downloading posts or images.

Options can also be set as environmental vars named like TUMBLR_IMPORTR_API_KEY.

Options:

  -api-key key
        Tumblr consumer API key
  -blog blog name
        blog name to import
  -image-dest path
        destination path to save images (default "images")
  -image-rewrites path
        path for JSON file containing image rewrites (default "image-rewrites.json")
  -image-url URL
        new base URL for images (default "/images")
  -max-conns-per-host int
        max number of simultaneous connections (default 9)
  -post-dest path
        destination path to save posts (default "post")
  -skip-api-request
        skip downloading posts from the API and just use -image-rewrites file to download images
  -skip-image-download
        skip downloading images (still rewrites)
  -timeout duration
        HTTP client timeout (default 10s)
  -workers int
        number of workers (default 9)
```

Pass the API key and blog name to tumblr-importr like

```bash
$ tumblr-importr -api-key '1234' -blog 'custom.blog.domain'
$ # Or
$ TUMBLR_IMPORTR_API_KEY='1234' TUMBLR_IMPORTR_BLOG='myblog' tumblr-importr
```

This will save all your posts and images in a format compatible with Hugo. Move the images into Hugo's static folder and then customize the sample layout so that the posts look like you want.

## To Do
- [ ] Better sample theme
- [ ] Better handling of cancellation
