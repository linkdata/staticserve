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
