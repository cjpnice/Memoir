package webassets

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var embedded embed.FS

// FS returns the embedded static web UI.
func FS() (fs.FS, error) {
	return fs.Sub(embedded, "static")
}
