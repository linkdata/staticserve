package staticserve

import (
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
)

// HandleFunc matches the signature of http.ServeMux.Handle().
//
// Handle and HandleFS pass method-aware patterns. Bare path patterns are normalized to GET.
type HandleFunc = func(uri string, handler http.Handler)

func escapeURIPath(fpath string) (string, error) {
	var parts []string
	for part := range strings.SplitSeq(fpath, "/") {
		parts = append(parts, url.PathEscape(part))
	}
	return url.JoinPath("/", parts...)
}

// Handle creates a new StaticServe for the fpath that returns the data given.
// Returns the URI of the resource.
func Handle(fpath string, data []byte, handleFn HandleFunc) (uri string, err error) {
	var ss *StaticServe
	if ss, err = New(fpath, data); err == nil {
		if uri, err = escapeURIPath(ss.Name); err == nil {
			handleFn(NormalizeGET(uri), ss)
		}
	}
	return
}

// HandleFS creates StaticServe handlers for the filepaths given.
// Returns the URI(s) of the resources. If an error occurs, the URI
// of the failed resource will be the empty string.
func HandleFS(fsys fs.FS, handleFn HandleFunc, root string, filepaths ...string) (uris []string, err error) {
	for _, filepath := range filepaths {
		var uri string
		b, ferr := readFSFile(fsys, root, filepath)
		if ferr == nil {
			uri, ferr = Handle(filepath, b, handleFn)
		}
		uris = append(uris, uri)
		err = errors.Join(err, ferr)
	}
	return
}
