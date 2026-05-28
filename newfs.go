package staticserve

import (
	"fmt"
	"io/fs"
	"strings"
)

func validateRelativeAssetPath(fpath string) error {
	if fpath == "." || !fs.ValidPath(fpath) {
		return fmt.Errorf("%w: %s", fs.ErrInvalid, fpath)
	}
	return nil
}

func validateAssetName(name string) error {
	return validateRelativeAssetPath(strings.TrimPrefix(name, "/"))
}

func readFSFile(fsys fs.FS, root, fpath string) (b []byte, err error) {
	if err = validateRelativeAssetPath(fpath); err == nil {
		if root == "" {
			root = "."
		}
		var sub fs.FS
		if sub, err = fs.Sub(fsys, root); err == nil {
			b, err = fs.ReadFile(sub, fpath)
		}
	}
	return
}

// NewFS reads the file at fpath from fsys and then calls New.
func NewFS(fsys fs.FS, root, fpath string) (ss *StaticServe, err error) {
	var b []byte
	if b, err = readFSFile(fsys, root, fpath); err == nil {
		ss, err = New(fpath, b)
	}
	return
}

// MustNewFS calls [NewFS] for each fpath relative to root and returns the
// resulting StaticServe values in order. It panics on the first error.
func MustNewFS(fsys fs.FS, root string, fpaths ...string) (ssl []*StaticServe) {
	ssl = make([]*StaticServe, 0, len(fpaths))
	for _, fpath := range fpaths {
		ss, err := NewFS(fsys, root, fpath)
		maybePanic(err)
		ssl = append(ssl, ss)
	}
	return
}
