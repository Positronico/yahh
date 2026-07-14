package histfmt

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// bash's HISTTIMEFORMAT timestamp comment: "#<epoch seconds>" on its own line.
var bashTimestampRe = regexp.MustCompile(`^#(\d{6,})$`)

// ParseBash parses a bash history file, with or without timestamp comments.
func ParseBash(data []byte) []Entry {
	var entries []Entry
	var ts time.Time
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		if m := bashTimestampRe.FindStringSubmatch(line); m != nil {
			n, _ := strconv.ParseInt(m[1], 10, 64)
			ts = time.Unix(n, 0)
			continue
		}
		entries = append(entries, Entry{When: ts, Cmd: line})
		ts = time.Time{}
	}
	return entries
}

// FormatBash writes plain command lines (no timestamp comments — they are
// only interpreted correctly when the reading shell has HISTTIMEFORMAT
// configured, so plain lines are the portable choice). Embedded newlines
// become "; " the way bash's cmdhist option flattens multiline commands.
func FormatBash(entries []Entry) []byte {
	var b bytes.Buffer
	for _, e := range entries {
		if e.Cmd == "" {
			continue
		}
		b.WriteString(strings.ReplaceAll(e.Cmd, "\n", "; "))
		b.WriteByte('\n')
	}
	return b.Bytes()
}
