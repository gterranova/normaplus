package assets

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:generate go run generate_assets.go

//go:embed all:dist
var embeddedFiles embed.FS

// GetFileSystem returns the static files for the frontend.
func GetFileSystem() http.FileSystem {
	f, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		// During initial development, the dist folder might be empty
		// We return an empty FS instead of panicking to allow the backend to build
		return http.FS(emptyFS{})
	}
	return http.FS(f)
}

type emptyFS struct{}

func (emptyFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrNotExist
}
