package histfmt

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// zsh's EXTENDED_HISTORY line: ": <start>:<elapsed>;<command>". The (?s)
// lets the command span the newlines restored from continuation lines.
var zshExtendedRe = regexp.MustCompile(`(?s)^: (\d+):(\d+);(.*)$`)

// zsh metafies bytes in [0x83, 0xa2] as Meta (0x83) followed by byte^0x20.
const metaChar = 0x83

// ParseZsh parses a zsh history file (plain or extended format, metafied).
func ParseZsh(data []byte) []Entry {
	text := string(unmetafy(data))
	var entries []Entry
	for _, item := range joinContinuations(strings.Split(text, "\n")) {
		if m := zshExtendedRe.FindStringSubmatch(item); m != nil {
			ts, _ := strconv.ParseInt(m[1], 10, 64)
			entries = append(entries, Entry{When: time.Unix(ts, 0), Cmd: m[3]})
		} else {
			entries = append(entries, Entry{Cmd: item})
		}
	}
	return entries
}

// FormatZsh writes entries in zsh's format: extended lines when a
// timestamp is known, plain lines otherwise (zsh reads both, regardless
// of the EXTENDED_HISTORY option). Embedded newlines become zsh's
// backslash-newline continuations and bytes are metafied.
func FormatZsh(entries []Entry) []byte {
	var b bytes.Buffer
	for _, e := range entries {
		if e.Cmd == "" {
			continue
		}
		cmd := strings.ReplaceAll(e.Cmd, "\n", "\\\n")
		line := cmd
		if !e.When.IsZero() {
			line = fmt.Sprintf(": %d:0;%s", e.When.Unix(), cmd)
		}
		b.Write(metafy([]byte(line)))
		b.WriteByte('\n')
	}
	return b.Bytes()
}

// joinContinuations rejoins physical lines whose trailing backslash marks
// an embedded newline, mirroring how zsh writes multiline commands.
func joinContinuations(lines []string) []string {
	var out []string
	cur := ""
	pending := false
	for _, line := range lines {
		if pending {
			cur += "\n" + line
		} else {
			cur = line
		}
		if strings.HasSuffix(line, "\\") {
			cur = cur[:len(cur)-1]
			pending = true
			continue
		}
		pending = false
		if cur != "" {
			out = append(out, cur)
		}
	}
	if pending && cur != "" {
		out = append(out, cur)
	}
	return out
}

func unmetafy(b []byte) []byte {
	if bytes.IndexByte(b, metaChar) < 0 {
		return b
	}
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] == metaChar && i+1 < len(b) {
			i++
			out = append(out, b[i]^0x20)
		} else {
			out = append(out, b[i])
		}
	}
	return out
}

func metafy(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for _, c := range b {
		if c >= 0x83 && c <= 0xa2 {
			out = append(out, metaChar, c^0x20)
		} else {
			out = append(out, c)
		}
	}
	return out
}
