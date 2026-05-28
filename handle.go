package staticserve

import (
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
)

// HandleFunc matches the signature of http.ServeMux.Handle().
type HandleFunc = func(uri string, handler http.Handler)

// escapeURIPath turns a slash-separated asset path into an absolute URI path
// with each segment percent-escaped. Segments are escaped individually with
// url.PathEscape (which escapes "/" and pattern-significant characters like
// '{') before url.JoinPath joins them under "/"; JoinPath only cleans and
// rejoins already-escaped segments, so this does not double-escape.
func escapeURIPath(fpath string) (string, error) {
	var parts []string
	for part := range strings.SplitSeq(fpath, "/") {
		parts = append(parts, url.PathEscape(part))
	}
	return url.JoinPath("/", parts...)
}

// Handle creates a new StaticServe for the fpath that returns the data given.
// Returns the URI of the resource. The pattern passed to handleFn is
// method-aware: bare path patterns are normalized to GET via [NormalizeGET].
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
	for _, fpath := range filepaths {
		var uri string
		b, ferr := readFSFile(fsys, root, fpath)
		if ferr == nil {
			uri, ferr = Handle(fpath, b, handleFn)
		}
		uris = append(uris, uri)
		err = errors.Join(err, ferr)
	}
	return
}
