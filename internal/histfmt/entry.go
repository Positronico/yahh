// Package histfmt parses and writes native zsh and bash history files.
package histfmt

import (
	"errors"
	"io/fs"
	"os"
	"time"
)

// Entry is one history event.
type Entry struct {
	When time.Time // zero when the source format carried no timestamp
	Cmd  string
}

// Supported shells.
const (
	ShellZsh  = "zsh"
	ShellBash = "bash"
)

// Parse decodes history data in the given shell's native format.
func Parse(data []byte, shell string) []Entry {
	if shell == ShellZsh {
		return ParseZsh(data)
	}
	return ParseBash(data)
}

// Format encodes entries in the given shell's native format.
func Format(entries []Entry, shell string) []byte {
	if shell == ShellZsh {
		return FormatZsh(entries)
	}
	return FormatBash(entries)
}

// ParseFile reads and parses a history file; a missing file yields no entries.
func ParseFile(path, shell string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return Parse(data, shell), nil
}

// AppendFile appends entries to a history file in the shell's native
// format, creating the file mode 0600 if needed. Appending (rather than
// rewriting) never clobbers lines other shells are adding concurrently.
func AppendFile(path, shell string, entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(Format(entries, shell))
	return err
}

// Detect guesses the shell format of history data. Only zsh's extended
// format (and metafied bytes) are distinguishable; plain command lines
// parse identically either way, so bash is the safe default.
func Detect(data []byte) string {
	for _, b := range data {
		if b == metaChar {
			return ShellZsh
		}
	}
	if zshExtendedRe.Match(data) {
		return ShellZsh
	}
	return ShellBash
}
