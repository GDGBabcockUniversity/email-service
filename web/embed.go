// Package web holds the admin dashboard that the service serves at /admin.
// Everything here is embedded into the binary, so the dashboard ships with the
// service and needs no separate build step or file layout at runtime.
package web

import (
	"embed"
	"io/fs"
)

//go:embed index.html static
var assets embed.FS

// Dashboard returns the dashboard page markup.
func Dashboard() ([]byte, error) {
	return assets.ReadFile("index.html")
}

// Static returns the dashboard's stylesheet, script, and image assets, rooted so
// that "css/dashboard.css" resolves the way the page references it.
func Static() fs.FS {
	static, err := fs.Sub(assets, "static")
	if err != nil {
		// Unreachable: the directory is embedded at compile time.
		panic(err)
	}
	return static
}
