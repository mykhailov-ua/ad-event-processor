package webstatic

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

func FS() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}

// DistFS is an alias for FS (embedded production/stub dist tree).
func DistFS() (fs.FS, error) {
	return FS()
}
