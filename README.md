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

Options can also be set as environmental variables named like
TUMBLR_IMPORTR_API_KEY. An API key must be created by registering at
<http://www.tumblr.com/oauth/apps>.

Downloading images can take a long time, so separate options exist for skipping
downloading posts or images. Image downloads will resume if restarted.

Options:

  -api-key key
        Tumblr consumer API key
  -blog blog name
        blog name to import
  -image-dest path
        destination path to save images (default "static/images")
  -image-rewrites path
        path for JSON file containing image rewrites (default "image-rewrites.json")
  -image-url URL
        new base URL for images (default "/images")
  -max-conns-per-host int
        max number of simultaneous connections (default 9)
  -post-dest path
        destination path to save posts (default "content/post")
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

## Philosophy

When converting a Tumblr blog to Hugo, you may initially think you want all your content converted to Markdown files. For example, you may think you want your link posts to become something like `### Link: <a href="$LINK">$TITLE</a>↵↵$CONTENT`. The trouble with this approach is that converting to Markdown loses formatting information from Tumblr and locks you into a single representation of the data which cannot be easily changed later.

How tumblr-importr works instead is it reads the common post metadata out of the Tumblr API (title, URL, slug, date, etc.) and writes that in the format Hugo expects, then it makes all of the other data from Tumblr on the post available as a custom parameter. Now you can format your link posts using Hugo's templating language to make it look exactly how you want:

```html
  <h3>Link: <a href="{{ .Params.tumblr.url }}">{{ .Params.tumblr.title }}</a></h3>

  {{ .Params.tumblr.description | safeHTML }}
```

If you decide the `H3` should be an `H2` or the content needs a wrapper `<div class="content">` or you want to change "Link:" to be an emoji 🔗, all you need to do is change your Hugo theme, rather than going back and reformatting all your Markdown files. All of the information that Tumblr had on the post is available, making it possible to fully replicate a Tumblr theme in Hugo without any information loss.

## To Do
- [ ] Better sample theme
- [ ] Better handling of cancellation
