package webdist

import (
	"embed"
	"io/fs"
)

// Files contains the committed Astro build output used by the shipped browser path.
//
//go:embed all:dist
var Files embed.FS

// DistFS returns the embedded Astro build output rooted at dist/.
func DistFS() (fs.FS, error) {
	return fs.Sub(Files, "dist")
}
