package staticserve

import (
	"fmt"
	"io/fs"
)

func readFSFile(fsys fs.FS, root, fpath string) (b []byte, err error) {
	if fpath == "." || !fs.ValidPath(fpath) {
		return nil, fmt.Errorf("%w: %s", fs.ErrInvalid, fpath)
	}
	if root == "" {
		root = "."
	}
	var sub fs.FS
	if sub, err = fs.Sub(fsys, root); err == nil {
		b, err = fs.ReadFile(sub, fpath)
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

func MustNewFS(fsys fs.FS, root string, fpaths ...string) (ssl []*StaticServe) {
	for _, fpath := range fpaths {
		ss, err := NewFS(fsys, root, fpath)
		MaybePanic(err)
		ssl = append(ssl, ss)
	}
	return
}
