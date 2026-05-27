package staticserve_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/linkdata/staticserve"
)

const someJavascript = `var jaws = null;

function jawsContains(a, v) {
	return a.indexOf(String(v).trim().toLowerCase()) !== -1;
}
`

func Test_New(t *testing.T) {
	ss, err := staticserve.New("test.js", []byte(someJavascript))
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(ss.ContentType, "javascript") {
		t.Error("ss not javascript")
	}
	ss2, err := staticserve.New("test.js.gz", ss.Gz)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(ss2.Gz, ss.Gz) {
		t.Error("bytes differ")
	}
	if !strings.Contains(ss2.ContentType, "javascript") {
		t.Error("ss2 not javascript")
	}
	ssUpper, err := staticserve.New("test.JS.gz", ss.Gz)
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(ssUpper.ContentType, "javascript") {
		t.Error("ssUpper not javascript")
	}
	if ss.Name != ss2.Name {
		t.Error(ss.Name, "!=", ss2.Name)
	}
	ss3, err := staticserve.New("test.foo123", nil)
	if err != nil {
		t.Error(err)
	}
	if ss3.ContentType != "" {
		t.Error(ss3.ContentType)
	}
}

func Test_New_PopulatesSize(t *testing.T) {
	data := []byte(someJavascript)
	ss, err := staticserve.New("test.js", data)
	if err != nil {
		t.Fatal(err)
	}
	if ss.Size != int64(len(data)) {
		t.Fatalf("plain input: expected size %d, got %d", len(data), ss.Size)
	}

	ssGz, err := staticserve.New("test.js.gz", ss.Gz)
	if err != nil {
		t.Fatal(err)
	}
	if ssGz.Size != int64(len(data)) {
		t.Fatalf(".gz input: expected size %d, got %d", len(data), ssGz.Size)
	}

	ssEmpty, err := staticserve.New("empty.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	if ssEmpty.Size != 0 {
		t.Fatalf("nil data: expected size 0, got %d", ssEmpty.Size)
	}
}

func Test_New_GZipIsComplete(t *testing.T) {
	data := []byte(someJavascript)
	ss, err := staticserve.New("test.js", data)
	if err != nil {
		t.Fatal(err)
	}
	if got := readGzip(t, ss.Gz); !bytes.Equal(got, data) {
		t.Fatalf("expected %q, got %q", data, got)
	}
}

func Test_New_RejectsInvalidGZipInput(t *testing.T) {
	ss, err := staticserve.New("test.js.gz", []byte("not gzip"))
	if err == nil {
		t.Fatal("expected error")
	}
	if ss != nil {
		t.Fatalf("expected nil StaticServe, got %#v", ss)
	}
}

func Test_New_RejectsTruncatedGZipInput(t *testing.T) {
	valid, err := staticserve.New("test.js", []byte(someJavascript))
	if err != nil {
		t.Fatal(err)
	}
	truncated := valid.Gz[:len(valid.Gz)-1]

	ss, err := staticserve.New("test.js.gz", truncated)
	if err == nil {
		t.Fatal("expected error")
	}
	if ss != nil {
		t.Fatalf("expected nil StaticServe, got %#v", ss)
	}
}

func Test_New_RejectsInvalidPath(t *testing.T) {
	for _, filename := range []string{".", "", "./file.txt", "dir/../file.txt", "dir//file.txt", "dir/./file.txt", "/file.txt"} {
		t.Run(filename, func(t *testing.T) {
			ss, err := staticserve.New(filename, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, fs.ErrInvalid) {
				t.Fatalf("expected fs.ErrInvalid, got %v", err)
			}
			if ss != nil {
				t.Fatalf("expected nil StaticServe, got %#v", ss)
			}
		})
	}
}

func Test_Must(t *testing.T) {
	ss := staticserve.Must("test", nil)
	if ss == nil {
		t.FailNow()
	}
}

func Test_MaybePanic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fail()
		}
	}()
	staticserve.MaybePanic(io.EOF)
	t.Fail()
}
