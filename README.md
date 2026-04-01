[![build](https://github.com/linkdata/staticserve/actions/workflows/build.yml/badge.svg)](https://github.com/linkdata/staticserve/actions/workflows/build.yml)
[![coverage](https://github.com/linkdata/staticserve/blob/coverage/main/badge.svg)](https://html-preview.github.io/?url=https://github.com/linkdata/staticserve/blob/coverage/main/report.html)
[![goreport](https://goreportcard.com/badge/github.com/linkdata/staticserve)](https://goreportcard.com/report/github.com/linkdata/staticserve)
[![Docs](https://godoc.org/github.com/linkdata/staticserve?status.svg)](https://godoc.org/github.com/linkdata/staticserve)

# staticserve

`staticserve` is a cache-busting HTTP handler for static files. It supports
GET operations requesting the file with no encoding or gzip encoding.

## Example

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/linkdata/staticserve"
)

func main() {
	mux := http.NewServeMux()
	uri, err := staticserve.Handle("app.js", []byte("console.log('hello');"), mux.Handle)
	if err != nil {
		log.Fatal(err)
	}

	// Use this URI in your HTML templates, e.g. <script src="{{.ScriptURI}}"></script>.
	fmt.Printf("script URI: %s\n", uri)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
```
