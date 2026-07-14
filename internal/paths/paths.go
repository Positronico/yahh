// Package paths centralizes filesystem locations and path normalization.
package paths

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DataDir returns the yahh data directory. Resolution order:
// $YAHH_DATA_DIR, $XDG_DATA_HOME/yahh, ~/.local/share/yahh.
func DataDir() string {
	if d := os.Getenv("YAHH_DATA_DIR"); d != "" {
		return filepath.Clean(d)
	}
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "yahh")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "yahh")
}

// NormalizeDir converts p to an absolute, symlink-resolved, cleaned path
// with no trailing slash. If the path no longer exists, symlinks are
// resolved for the longest existing ancestor and the remainder is joined
// lexically, so vanished realm directories still normalize consistently.
func NormalizeDir(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return trimTrailingSlash(resolved), nil
	}
	dir := filepath.Clean(abs)
	rest := ""
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		if rest == "" {
			rest = filepath.Base(dir)
		} else {
			rest = filepath.Join(filepath.Base(dir), rest)
		}
		dir = parent
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			return trimTrailingSlash(filepath.Join(resolved, rest)), nil
		}
	}
	return trimTrailingSlash(filepath.Clean(abs)), nil
}

func trimTrailingSlash(p string) string {
	if len(p) > 1 {
		p = strings.TrimSuffix(p, string(filepath.Separator))
	}
	return p
}

var invalidNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// SanitizeName reduces s to a filesystem- and display-safe realm name:
// only [A-Za-z0-9._-], no leading dots or dashes, at most 40 characters.
func SanitizeName(s string) string {
	s = invalidNameChars.ReplaceAllString(s, "-")
	s = strings.TrimLeft(s, "-.")
	if len(s) > 40 {
		s = s[:40]
	}
	if s == "" {
		s = "realm"
	}
	return s
}
