// Package version reports which build is running, using the VCS metadata the
// Go toolchain embeds in every binary. Nothing needs to be passed at build
// time.
package version

import (
	"runtime/debug"
	"sync"
)

// override, when set, is reported as-is. Builds that cannot see the Git
// repository (the Dockerfile, for one) set it with:
//
//	-ldflags "-X github.com/dunamismax/go-web-server/internal/version.override=v1.2.3"
var override string

// String returns the module version for tagged builds (for example
// "v5.0.0") and otherwise the short commit hash, with "-dirty" appended when
// the working tree had uncommitted changes. It returns "dev" when no build
// information is available, as under "go test".
var String = sync.OnceValue(func() string {
	if override != "" {
		return override
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	var revision string
	var dirty bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}

	if v := info.Main.Version; v != "" && v != "(devel)" && !isPseudoVersion(v) {
		return v
	}
	if revision == "" {
		return "dev"
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	if dirty {
		revision += "-dirty"
	}
	return revision
})

// isPseudoVersion reports whether v looks like the "v0.0.0-<time>-<hash>"
// form Go synthesizes for untagged commits, which is less readable than the
// bare hash.
func isPseudoVersion(v string) bool {
	dashes := 0
	for _, c := range v {
		if c == '-' {
			dashes++
		}
	}
	return dashes >= 2
}
