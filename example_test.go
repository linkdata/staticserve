package staticserve_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/linkdata/staticserve"
)

func ExampleHandle() {
	mux := http.NewServeMux()
	uri, err := staticserve.Handle("app.js", []byte("console.log('hello');"), mux.Handle)
	if err != nil {
		panic(err)
	}

	req := httptest.NewRequest(http.MethodGet, uri, nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	res := rr.Result()

	fmt.Println(res.StatusCode == http.StatusOK)
	fmt.Println(res.Header.Get("Content-Encoding"))
	fmt.Println(strings.HasPrefix(uri, "/app."))
	fmt.Println(strings.HasSuffix(uri, ".js"))
	// Output:
	// true
	// gzip
	// true
	// true
}
