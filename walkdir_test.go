package staticserve_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/linkdata/staticserve"
)

func Test_WalkDir(t *testing.T) {
	filepaths := assetFilepaths(t, assetsFS, "assets")
	expected := expectedStaticAssetMap(t, assetsFS, "assets", "/", filepaths...)
	seen := map[string]bool{}
	mux := http.NewServeMux()
	err := staticserve.WalkDir(assetsFS, "assets", func(filepath string, ss *staticserve.StaticServe) (err error) {
		exp, ok := expected[filepath]
		if !ok {
			t.Errorf("unexpected filepath: %q", filepath)
			return
		}
		if ss == nil {
			t.Errorf("nil StaticServe for %q", filepath)
			return
		}
		if ss.Name != exp.name {
			t.Errorf("%q: expected name %q, got %q", filepath, exp.name, ss.Name)
		}
		if ss.ContentType != exp.contentType {
			t.Errorf("%q: expected content type %q, got %q", filepath, exp.contentType, ss.ContentType)
		}
		if len(ss.Gz) == 0 {
			t.Errorf("%q: empty gz payload", filepath)
		}
		if !bytes.Equal(ss.Gz, exp.gz) {
			t.Errorf("%q: gz payload mismatch", filepath)
		}

		uri := "/static/" + ss.Name
		t.Log(filepath, uri)
		seen[filepath] = true
		mux.Handle("GET "+uri, ss)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != len(expected) {
		t.Fatalf("expected %d files, got %d", len(expected), len(seen))
	}

	for filepath, exp := range expected {
		if !seen[filepath] {
			t.Fatalf("missing filepath %q", filepath)
		}
		uri := "/static/" + exp.name

		rq := httptest.NewRequest(http.MethodGet, uri, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, rq)
		res := rr.Result()
		if sc := res.StatusCode; sc != http.StatusOK {
			t.Errorf("%q plain: expected status %d, got %d", filepath, http.StatusOK, sc)
		}
		if cc := res.Header.Get("Cache-Control"); cc != staticserve.HeaderCacheControl[0] {
			t.Errorf("%q plain: unexpected cache-control %q", filepath, cc)
		}
		if vary := res.Header.Get("Vary"); vary != staticserve.HeaderVary[0] {
			t.Errorf("%q plain: unexpected vary %q", filepath, vary)
		}
		if ce := res.Header.Get("Content-Encoding"); ce != "" {
			t.Errorf("%q plain: unexpected content-encoding %q", filepath, ce)
		}
		if ct := res.Header.Get("Content-Type"); ct != exp.contentType {
			t.Errorf("%q plain: expected content type %q, got %q", filepath, exp.contentType, ct)
		}
		b, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, exp.plain) {
			t.Errorf("%q plain: unexpected body", filepath)
		}
		if err = res.Body.Close(); err != nil {
			t.Fatal(err)
		}

		rq = httptest.NewRequest(http.MethodGet, uri, nil)
		rq.Header.Set("Accept-Encoding", "gzip")
		rr = httptest.NewRecorder()
		mux.ServeHTTP(rr, rq)
		res = rr.Result()
		if sc := res.StatusCode; sc != http.StatusOK {
			t.Errorf("%q gzip: expected status %d, got %d", filepath, http.StatusOK, sc)
		}
		if cc := res.Header.Get("Cache-Control"); cc != staticserve.HeaderCacheControl[0] {
			t.Errorf("%q gzip: unexpected cache-control %q", filepath, cc)
		}
		if vary := res.Header.Get("Vary"); vary != staticserve.HeaderVary[0] {
			t.Errorf("%q gzip: unexpected vary %q", filepath, vary)
		}
		if ce := res.Header.Get("Content-Encoding"); ce != "gzip" {
			t.Errorf("%q gzip: expected content-encoding %q, got %q", filepath, "gzip", ce)
		}
		if cl := res.Header.Get("Content-Length"); cl != strconv.Itoa(len(exp.gz)) {
			t.Errorf("%q gzip: expected content-length %d, got %q", filepath, len(exp.gz), cl)
		}
		if ct := res.Header.Get("Content-Type"); ct != exp.contentType {
			t.Errorf("%q gzip: expected content type %q, got %q", filepath, exp.contentType, ct)
		}
		b, err = io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, exp.gz) {
			t.Errorf("%q gzip: unexpected body", filepath)
		}
		if err = res.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func Test_WalkDir_EmptyRoot(t *testing.T) {
	fsys := fstest.MapFS{
		"file.txt": {Data: []byte("hello")},
	}

	var filenames []string
	err := staticserve.WalkDir(fsys, "", func(filename string, ss *staticserve.StaticServe) error {
		filenames = append(filenames, filename)
		if ss == nil {
			t.Fatal("nil StaticServe")
		}
		if got := readGzip(t, ss.Gz); !bytes.Equal(got, []byte("hello")) {
			t.Fatalf("expected file contents, got %q", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(filenames) != 1 || filenames[0] != "file.txt" {
		t.Fatalf("expected [file.txt], got %v", filenames)
	}
}

func Test_WalkDir_RootDotPreservesTopLevelDotPath(t *testing.T) {
	fsys := fstest.MapFS{
		".well-known/acme.txt": {Data: []byte("challenge")},
	}

	var gotFilename, gotName string
	err := staticserve.WalkDir(fsys, ".", func(filename string, ss *staticserve.StaticServe) error {
		gotFilename = filename
		gotName = ss.Name
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotFilename != ".well-known/acme.txt" {
		t.Fatalf("expected dot path to be preserved, got %q", gotFilename)
	}
	if !strings.HasPrefix(gotName, ".well-known/acme.") || !strings.HasSuffix(gotName, ".txt") {
		t.Fatalf("expected cache-busted dot path, got %q", gotName)
	}
}
