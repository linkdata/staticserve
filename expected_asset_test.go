package staticserve_test

import (
	"bytes"
	"compress/gzip"
	"embed"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/linkdata/staticserve"
)

//go:embed assets
var assetsFS embed.FS

type expectedStaticAsset struct {
	filepath    string
	name        string
	uri         string
	contentType string
	plain       []byte
	gz          []byte
	ss          *staticserve.StaticServe
}

func readGzip(t *testing.T, b []byte) []byte {
	t.Helper()
	gzr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := io.ReadAll(gzr)
	if cerr := gzr.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		t.Fatal(err)
	}
	return plain
}

func assetFilepaths(t *testing.T, fsys fs.FS, root string) (filepaths []string) {
	t.Helper()
	root = path.Clean(root)
	err := fs.WalkDir(fsys, root, func(pathname string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		filepaths = append(filepaths, strings.TrimPrefix(pathname, root+"/"))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(filepaths)
	if len(filepaths) == 0 {
		t.Fatal("expected at least one asset file")
	}
	return
}

func expectedStaticAssets(t *testing.T, fsys fs.FS, root, uriPrefix string, filepaths ...string) (expected []expectedStaticAsset) {
	t.Helper()
	if len(filepaths) == 0 {
		filepaths = assetFilepaths(t, fsys, root)
	}
	for _, filepath := range filepaths {
		b, err := fs.ReadFile(fsys, path.Join(root, filepath))
		if err != nil {
			t.Fatal(err)
		}
		ss, err := staticserve.New(filepath, b)
		if err != nil {
			t.Fatal(err)
		}
		plain := b
		if strings.HasSuffix(filepath, ".gz") {
			plain = readGzip(t, b)
		}
		expected = append(expected, expectedStaticAsset{
			filepath:    filepath,
			name:        ss.Name,
			uri:         path.Join(uriPrefix, ss.Name),
			contentType: ss.ContentType,
			plain:       plain,
			gz:          ss.Gz,
			ss:          ss,
		})
	}
	return
}

func expectedStaticAssetMap(t *testing.T, fsys fs.FS, root, uriPrefix string, filepaths ...string) map[string]expectedStaticAsset {
	t.Helper()
	expected := map[string]expectedStaticAsset{}
	for _, exp := range expectedStaticAssets(t, fsys, root, uriPrefix, filepaths...) {
		expected[exp.filepath] = exp
	}
	return expected
}
