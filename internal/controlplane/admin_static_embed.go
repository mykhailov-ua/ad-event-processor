package controlplane

import (
	webstatic "ad-event-processor/web"
	"io/fs"
)

func adminStaticFS() (fs.FS, error) {
	return webstatic.FS()
}
