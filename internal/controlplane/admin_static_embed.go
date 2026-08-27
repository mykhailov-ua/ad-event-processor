package controlplane

import (
	"embed"
	"io/fs"
)

//go:embed admin_static_stub/*
var adminStaticEmbed embed.FS

func adminStaticFS() (fs.FS, error) {
	return fs.Sub(adminStaticEmbed, "admin_static_stub")
}
