package staticserve

import (
	"bytes"
	"compress/gzip"
	"errors"
	"hash/fnv"
	"io"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
)

// StaticServe is an http.Handler that serves a single asset under a
// cache-busted file name. The asset is held gzip-compressed in memory and
// served either as gzip (when the client advertises Accept-Encoding: gzip)
// or decompressed on the fly otherwise.
//
// Instances are safe for concurrent use after construction and are intended
// to be created via [New], [NewFS], [Must], or [MustNewFS] so that all fields
// are populated consistently.
type StaticServe struct {
	Name        string // the cache-busting file name, e.g. "static/filename.1234567.js"
	ContentType string // Content-Type of the file, e.g. "application/javascript"
	Size        int64  // uncompressed length of the asset in bytes
	Gz          []byte // gzipped data, will be unpacked as needed
}

// New returns a StaticServe that serves the given data with a filename like 'filename.12345678.ext'.
// The filename must be a valid slash-separated relative path, excluding ".".
// The filename must have the suffix ".gz" if the data is GZip compressed. The ".gz" suffix will
// not be part of the filename presented in this case.
func New(filename string, data []byte) (ss *StaticServe, err error) {
	if err = validateAssetPath(filename); err == nil {
		var gz []byte
		var size int64
		// The cache-busting name is derived from a hash of the uncompressed
		// content, so it stays stable regardless of how the data was compressed
		// (e.g. across gzip level or Go toolchain changes).
		h := fnv.New64a()
		if strings.HasSuffix(filename, ".gz") {
			gz = append([]byte(nil), data...)
			filename = strings.TrimSuffix(filename, ".gz")
			var gzr *gzip.Reader
			if gzr, err = gzip.NewReader(bytes.NewReader(gz)); err == nil {
				size, err = io.Copy(h, gzr) // #nosec G110
				err = errors.Join(err, gzr.Close())
			}
		} else {
			size = int64(len(data))
			_, _ = h.Write(data) // hash.Hash.Write never returns an error
			var buf bytes.Buffer
			gzw := gzip.NewWriter(&buf)
			_, err = gzw.Write(data)
			if err = errors.Join(err, gzw.Close()); err == nil {
				gz = buf.Bytes()
			}
		}

		if err == nil {
			ext := filepath.Ext(filename)
			filename = strings.TrimSuffix(filename, ext)
			ss = &StaticServe{
				Name:        filename + "." + strconv.FormatUint(h.Sum64(), 36) + ext,
				ContentType: mime.TypeByExtension(ext),
				Size:        size,
				Gz:          gz,
			}
		}
	}
	return
}

// MaybePanic panics if err is non-nil. It is used by [Must] and [MustNewFS]
// to convert initialization errors into panics.
func MaybePanic(err error) {
	if err != nil {
		panic(err)
	}
}

// Must calls New and panics on error.
func Must(filename string, data []byte) (ss *StaticServe) {
	var err error
	ss, err = New(filename, data)
	MaybePanic(err)
	return
}
