package staticserve

import (
	"bytes"
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

// HeaderCacheControl is the Cache-Control header value sent with successful
// responses. Its default marks the asset as immutable for one year, which is
// safe because the served file name is cache-busted via a content hash.
var HeaderCacheControl = []string{"public, max-age=31536000, s-maxage=31536000, immutable"}

// HeaderVary is the Vary header value sent with successful responses.
var HeaderVary = []string{"Accept-Encoding"}

// HeaderAllow is the Allow header value sent with 405 Method Not Allowed
// responses to requests that use methods other than GET or HEAD.
var HeaderAllow = []string{http.MethodGet + ", " + http.MethodHead}

var headerContentEncoding = []string{"gzip"}

func acceptsGzip(hdr http.Header) bool {
	for _, value := range hdr.Values("Accept-Encoding") {
		for encoding := range strings.SplitSeq(value, ",") {
			coding, params, err := mime.ParseMediaType(encoding)
			if err != nil || !strings.EqualFold(coding, "gzip") {
				continue
			}
			if q, ok := params["q"]; ok {
				qvalue, err := strconv.ParseFloat(q, 64)
				if err == nil && qvalue <= 0 {
					continue
				}
			}
			return true
		}
	}
	return false
}

// ServeHTTP serves the asset for GET and HEAD requests.
//
// When the client advertises Accept-Encoding: gzip the gzip-compressed bytes
// are served verbatim; otherwise the asset is decompressed on the fly.
// Content-Length always reflects the size of the body that would be returned
// by an equivalent GET, so HEAD responses are usable for size discovery.
//
// Methods other than GET and HEAD receive 405 Method Not Allowed with an
// Allow header. If the stored gzip stream cannot be opened (which should not
// happen for instances created via [New]), 500 Internal Server Error is
// returned with no body.
func (ss *StaticServe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body io.Reader
	statusCode := http.StatusMethodNotAllowed
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		hdr := w.Header()
		if acceptsGzip(r.Header) {
			body = bytes.NewReader(ss.Gz)
			hdr["Content-Encoding"] = headerContentEncoding
			hdr["Content-Length"] = []string{strconv.Itoa(len(ss.Gz))}
		} else {
			statusCode = http.StatusInternalServerError
			if gzr, err := gzip.NewReader(bytes.NewReader(ss.Gz)); err == nil {
				defer func() { _ = gzr.Close() }()
				body = gzr
				hdr["Content-Length"] = []string{strconv.FormatInt(ss.Size, 10)}
			}
		}
		if body != nil {
			statusCode = http.StatusOK
			hdr["Cache-Control"] = HeaderCacheControl
			hdr["Vary"] = HeaderVary
			if ss.ContentType != "" {
				hdr["Content-Type"] = []string{ss.ContentType}
			}
		}
	} else {
		w.Header()["Allow"] = HeaderAllow
	}
	w.WriteHeader(statusCode)
	if body != nil && r.Method != http.MethodHead {
		_, _ = io.Copy(w, body)
	}
}
