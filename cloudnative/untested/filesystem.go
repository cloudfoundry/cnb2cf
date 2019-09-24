package untested

import (
	"net/http"

	"github.com/rakyll/statik/fs"
)

type Filesystem struct {
	fs http.FileSystem
}

func NewFilesystem(fs http.FileSystem) Filesystem {
	return Filesystem{
		fs: fs,
	}
}

func (f Filesystem) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(f.fs, name)
}
