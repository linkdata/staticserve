package staticserve

import (
	"io/fs"
	"strings"
)

// WalkDir walks the file tree rooted at root, calling fn for each file in the tree with
// the filename having root trimmed (e.g. "root/dir/file.ext" becomes "dir/file.ext").
func WalkDir(fsys fs.FS, root string, fn func(filename string, ss *StaticServe) (err error)) (err error) {
	if root == "" {
		root = "."
	}
	err = fs.WalkDir(fsys, root, func(filename string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			var b []byte
			if b, err = fs.ReadFile(fsys, filename); err == nil {
				var ss *StaticServe
				filename = strings.TrimPrefix(filename, root+"/")
				if ss, err = New(filename, b); err == nil {
					err = fn(filename, ss)
				}
			}
		}
		return err
	})
	return
}
