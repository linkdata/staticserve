package staticserve_test

import (
	"bytes"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/linkdata/staticserve"
)

func TestNewFS(t *testing.T) {
	for _, exp := range expectedStaticAssets(t, assetsFS, "assets", "/") {
		ss, err := staticserve.NewFS(assetsFS, "assets", exp.filepath)
		if err != nil {
			t.Fatal(err)
		}
		if ss == nil {
			t.Fatalf("%q: nil StaticServe", exp.filepath)
		}
		if ss.Name != exp.name {
			t.Errorf("%q: expected name %q, got %q", exp.filepath, exp.name, ss.Name)
		}
		if ss.ContentType != exp.contentType {
			t.Errorf("%q: expected content type %q, got %q", exp.filepath, exp.contentType, ss.ContentType)
		}
		if !bytes.Equal(ss.Gz, exp.gz) {
			t.Errorf("%q: gz payload mismatch", exp.filepath)
		}
	}
}

func TestNewFS_RejectsRootEscape(t *testing.T) {
	fsys := fstest.MapFS{
		"assets/public.txt": {Data: []byte("public")},
		"secret.txt":        {Data: []byte("secret")},
	}
	ss, err := staticserve.NewFS(fsys, "assets", "../secret.txt")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("expected fs.ErrInvalid, got %v", err)
	}
	if ss != nil {
		t.Fatalf("expected nil StaticServe, got %#v", ss)
	}
}

func TestNewFS_RejectsAbsolutePath(t *testing.T) {
	fsys := fstest.MapFS{
		"assets/public.txt": {Data: []byte("public")},
	}
	ss, err := staticserve.NewFS(fsys, "assets", "/public.txt")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("expected fs.ErrInvalid, got %v", err)
	}
	if ss != nil {
		t.Fatalf("expected nil StaticServe, got %#v", ss)
	}
}

func TestNewFS_RejectsInvalidRoot(t *testing.T) {
	fsys := fstest.MapFS{
		"assets/public.txt": {Data: []byte("public")},
	}
	ss, err := staticserve.NewFS(fsys, "../assets", "public.txt")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("expected fs.ErrInvalid, got %v", err)
	}
	if ss != nil {
		t.Fatalf("expected nil StaticServe, got %#v", ss)
	}
}

func TestNewFS_EmptyRoot(t *testing.T) {
	fsys := fstest.MapFS{
		"assets/public.txt": {Data: []byte("public")},
	}
	ss, err := staticserve.NewFS(fsys, "", "assets/public.txt")
	if err != nil {
		t.Fatal(err)
	}
	if ss == nil {
		t.Fatal("nil StaticServe")
	}
	if ss.Name == "" {
		t.Fatal("empty name")
	}
	if got := readGzip(t, ss.Gz); !bytes.Equal(got, []byte("public")) {
		t.Fatalf("expected public asset, got %q", got)
	}
}

func TestMustNewFS(t *testing.T) {
	expected := expectedStaticAssets(t, assetsFS, "assets", "/")
	filepaths := make([]string, 0, len(expected))
	for _, exp := range expected {
		filepaths = append(filepaths, exp.filepath)
	}

	got := staticserve.MustNewFS(assetsFS, "assets", filepaths...)
	if len(got) != len(expected) {
		t.Fatalf("expected %d StaticServe values, got %d", len(expected), len(got))
	}
	for i := range expected {
		if got[i] == nil {
			t.Fatalf("%q: nil StaticServe", expected[i].filepath)
		}
		if got[i].Name != expected[i].name {
			t.Errorf("%q: expected name %q, got %q", expected[i].filepath, expected[i].name, got[i].Name)
		}
		if got[i].ContentType != expected[i].contentType {
			t.Errorf("%q: expected content type %q, got %q", expected[i].filepath, expected[i].contentType, got[i].ContentType)
		}
		if !bytes.Equal(got[i].Gz, expected[i].gz) {
			t.Errorf("%q: gz payload mismatch", expected[i].filepath)
		}
	}
}
