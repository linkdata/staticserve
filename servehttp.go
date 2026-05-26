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

var HeaderCacheControl = []string{"public, max-age=31536000, s-maxage=31536000, immutable"}
var HeaderVary = []string{"Accept-Encoding"}
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
				defer gzr.Close()
				body = gzr
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
	}
	w.WriteHeader(statusCode)
	if body != nil && r.Method != http.MethodHead {
		_, _ = io.Copy(w, body)
	}
}
