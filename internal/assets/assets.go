// Package assets serves the static files that ship inside the binary: the
// compiled stylesheet, htmx, and the favicon.
//
// static/app.css is generated from tailwind.css by "make css".
package assets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"
)

//go:embed static
var embedded embed.FS

// URLPrefix is where the files are mounted.
const URLPrefix = "/static/"

// liveDir is where the static files live in a source checkout, relative to
// the repository root.
const liveDir = "internal/assets/static"

var (
	mu       sync.RWMutex
	files    = mustSub(embedded, "static")
	live     bool
	versions = map[string]string{}
)

// UseLiveFiles switches from the embedded files to the ones on disk when the
// working directory is a source checkout. Stylesheet edits then show up on a
// plain browser refresh, without rebuilding the binary. It reports whether
// the switch happened.
func UseLiveFiles() bool {
	if info, err := os.Stat(liveDir); err != nil || !info.IsDir() {
		return false
	}

	mu.Lock()
	defer mu.Unlock()

	files = os.DirFS(liveDir)
	live = true
	versions = map[string]string{}
	return true
}

// Path returns the URL for a static file, such as "app.css". Embedded files
// carry a content hash in the query string so browsers can cache them
// forever and still pick up a new version after a deploy.
func Path(name string) string {
	mu.RLock()
	isLive := live
	version, cached := versions[name]
	mu.RUnlock()

	if isLive {
		return URLPrefix + name
	}
	if !cached {
		version = hash(name)
		mu.Lock()
		versions[name] = version
		mu.Unlock()
	}
	if version == "" {
		return URLPrefix + name
	}
	return URLPrefix + name + "?v=" + version
}

// Handler serves the static files under URLPrefix.
func Handler() http.Handler {
	mu.RLock()
	fsys, isLive := files, live
	mu.RUnlock()

	server := http.StripPrefix(URLPrefix, http.FileServerFS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			// No directory listings.
			http.NotFound(w, r)
			return
		}

		switch {
		case isLive:
			w.Header().Set("Cache-Control", "no-cache")
		case r.URL.Query().Has("v"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		default:
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		server.ServeHTTP(w, r)
	})
}

// Favicon serves static/favicon.ico for the conventional /favicon.ico URL.
func Favicon() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		fsys := files
		mu.RUnlock()

		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeFileFS(w, r, fsys, "favicon.ico")
	})
}

func hash(name string) string {
	mu.RLock()
	fsys := files
	mu.RUnlock()

	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:6])
}

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
