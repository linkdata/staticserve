package staticserve_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/linkdata/staticserve"
)

func TestHandle_Pattern(t *testing.T) {
	var gotPattern string
	uri, err := staticserve.Handle("file.txt", []byte("abc"), func(pattern string, _ http.Handler) {
		gotPattern = pattern
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "GET " + uri; gotPattern != want {
		t.Fatalf("expected pattern %q, got %q", want, gotPattern)
	}
}

func TestHandle_EscapesSpecialFilenameURI(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Handle panicked: %v", r)
		}
	}()

	mux := http.NewServeMux()
	uri, err := staticserve.Handle("dir/{slug} file#query?percent%.txt", []byte("abc"), mux.Handle)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"{", "}", " ", "#", "?", "%."} {
		if strings.Contains(uri, raw) {
			t.Fatalf("expected escaped uri, got %q", uri)
		}
	}
	for _, escaped := range []string{"%7Bslug%7D", "%20", "%23", "%3F", "%25"} {
		if !strings.Contains(uri, escaped) {
			t.Fatalf("expected uri %q to contain %q", uri, escaped)
		}
	}

	rq := httptest.NewRequest(http.MethodGet, uri, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, rq)
	if sc := rr.Result().StatusCode; sc != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, sc)
	}
}

func TestHandle_DoesNotRegisterWildcardRoute(t *testing.T) {
	mux := http.NewServeMux()
	uri, err := staticserve.Handle("dir/{asset}/file.txt", []byte("abc"), mux.Handle)
	if err != nil {
		t.Fatal(err)
	}

	rq := httptest.NewRequest(http.MethodGet, uri, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, rq)
	if sc := rr.Result().StatusCode; sc != http.StatusOK {
		t.Fatalf("literal uri: expected status %d, got %d", http.StatusOK, sc)
	}

	wildcardCandidate := strings.Replace(uri, "%7Basset%7D", "other", 1)
	wildcardCandidate = strings.Replace(wildcardCandidate, "{asset}", "other", 1)
	rq = httptest.NewRequest(http.MethodGet, wildcardCandidate, nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, rq)
	if sc := rr.Result().StatusCode; sc != http.StatusNotFound {
		t.Fatalf("wildcard candidate %q: expected status %d, got %d", wildcardCandidate, http.StatusNotFound, sc)
	}
}

func TestHandle_RejectsInvalidPath(t *testing.T) {
	for _, fpath := range []string{".", "./file.txt", "dir/../file.txt", "dir//file.txt", "dir/./file.txt", "/file.txt"} {
		t.Run(fpath, func(t *testing.T) {
			called := false
			uri, err := staticserve.Handle(fpath, []byte("abc"), func(string, http.Handler) {
				called = true
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, fs.ErrInvalid) {
				t.Fatalf("expected fs.ErrInvalid, got %v", err)
			}
			if uri != "" {
				t.Fatalf("expected empty uri, got %q", uri)
			}
			if called {
				t.Fatal("handler was registered")
			}
		})
	}
}

func TestHandleFS_RejectsRootEscape(t *testing.T) {
	fsys := fstest.MapFS{
		"assets/public.txt": {Data: []byte("public")},
		"secret.txt":        {Data: []byte("secret")},
	}
	var handled []string
	uris, err := staticserve.HandleFS(fsys, func(pattern string, _ http.Handler) {
		handled = append(handled, pattern)
	}, "assets", "public.txt", "../secret.txt")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("expected fs.ErrInvalid, got %v", err)
	}
	if len(uris) != 2 {
		t.Fatalf("expected 2 uris, got %d", len(uris))
	}
	if uris[0] == "" {
		t.Fatal("expected public asset uri")
	}
	if uris[1] != "" {
		t.Fatalf("expected empty uri for invalid path, got %q", uris[1])
	}
	if len(handled) != 1 {
		t.Fatalf("expected one handled route, got %d: %v", len(handled), handled)
	}
}

func TestHandleFS(t *testing.T) {
	filepaths := assetFilepaths(t, assetsFS, "assets")
	expected := expectedStaticAssets(t, assetsFS, "assets", "/", filepaths...)

	mux := http.NewServeMux()
	uris, err := staticserve.HandleFS(assetsFS, mux.Handle, "assets", filepaths...)
	if err != nil {
		t.Fatal(err)
	}
	if len(uris) != len(expected) {
		t.Fatal(len(uris))
	}

	for i, exp := range expected {
		if uris[i] != exp.uri {
			t.Errorf("%q: expected uri %q, got %q", exp.filepath, exp.uri, uris[i])
		}

		rq := httptest.NewRequest(http.MethodGet, exp.uri, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, rq)
		res := rr.Result()
		if sc := res.StatusCode; sc != http.StatusOK {
			t.Errorf("%q plain: expected status %d, got %d", exp.filepath, http.StatusOK, sc)
		}
		if cc := res.Header.Get("Cache-Control"); cc != staticserve.HeaderCacheControl[0] {
			t.Errorf("%q plain: expected cache-control %q, got %q", exp.filepath, staticserve.HeaderCacheControl[0], cc)
		}
		if vary := res.Header.Get("Vary"); vary != staticserve.HeaderVary[0] {
			t.Errorf("%q plain: expected vary %q, got %q", exp.filepath, staticserve.HeaderVary[0], vary)
		}
		if ce := res.Header.Get("Content-Encoding"); ce != "" {
			t.Errorf("%q plain: expected empty content-encoding, got %q", exp.filepath, ce)
		}
		if ct := res.Header.Get("Content-Type"); ct != exp.contentType {
			t.Errorf("%q plain: expected content type %q, got %q", exp.filepath, exp.contentType, ct)
		}
		b, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, exp.plain) {
			t.Errorf("%q plain: body mismatch", exp.filepath)
		}
		if err := res.Body.Close(); err != nil {
			t.Fatal(err)
		}

		rq = httptest.NewRequest(http.MethodGet, exp.uri, nil)
		rq.Header.Set("Accept-Encoding", "gzip")
		rr = httptest.NewRecorder()
		mux.ServeHTTP(rr, rq)
		res = rr.Result()
		if sc := res.StatusCode; sc != http.StatusOK {
			t.Errorf("%q gzip: expected status %d, got %d", exp.filepath, http.StatusOK, sc)
		}
		if cc := res.Header.Get("Cache-Control"); cc != staticserve.HeaderCacheControl[0] {
			t.Errorf("%q gzip: expected cache-control %q, got %q", exp.filepath, staticserve.HeaderCacheControl[0], cc)
		}
		if vary := res.Header.Get("Vary"); vary != staticserve.HeaderVary[0] {
			t.Errorf("%q gzip: expected vary %q, got %q", exp.filepath, staticserve.HeaderVary[0], vary)
		}
		if ce := res.Header.Get("Content-Encoding"); ce != "gzip" {
			t.Errorf("%q gzip: expected content-encoding %q, got %q", exp.filepath, "gzip", ce)
		}
		if cl := res.Header.Get("Content-Length"); cl != strconv.Itoa(len(exp.gz)) {
			t.Errorf("%q gzip: expected content-length %d, got %q", exp.filepath, len(exp.gz), cl)
		}
		if ct := res.Header.Get("Content-Type"); ct != exp.contentType {
			t.Errorf("%q gzip: expected content type %q, got %q", exp.filepath, exp.contentType, ct)
		}
		b, err = io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, exp.gz) {
			t.Errorf("%q gzip: body mismatch", exp.filepath)
		}
		if err := res.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
