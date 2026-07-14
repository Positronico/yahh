package histfmt

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestParseZshExtended(t *testing.T) {
	data := []byte(": 1700000000:0;git status\n: 1700000005:2;make test\nplain command\n")
	got := ParseZsh(data)
	want := []Entry{
		{When: time.Unix(1700000000, 0), Cmd: "git status"},
		{When: time.Unix(1700000005, 0), Cmd: "make test"},
		{Cmd: "plain command"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestZshMultilineRoundTrip(t *testing.T) {
	entries := []Entry{
		{When: time.Unix(1700000000, 0), Cmd: "for f in *; do\n  echo $f\ndone"},
		{When: time.Unix(1700000001, 0), Cmd: "echo after"},
	}
	got := ParseZsh(FormatZsh(entries))
	if !reflect.DeepEqual(got, entries) {
		t.Errorf("round trip:\n got %+v\nwant %+v", got, entries)
	}
}

func TestZshMetafiedRoundTrip(t *testing.T) {
	// "Ã" is UTF-8 0xC3 0x83; the 0x83 byte collides with zsh's Meta
	// character and must be escaped on write and restored on read.
	entries := []Entry{{When: time.Unix(1700000000, 0), Cmd: "echo Ã"}}
	formatted := FormatZsh(entries)
	if !bytes.Contains(formatted, []byte{0x83, 0x83 ^ 0x20}) {
		t.Errorf("0x83 byte not metafied in %q", formatted)
	}
	got := ParseZsh(formatted)
	if !reflect.DeepEqual(got, entries) {
		t.Errorf("round trip: got %+v, want %+v", got, entries)
	}
}

func TestParseZshRealMetafiedFile(t *testing.T) {
	// As zsh itself would write "echo Ã" (0xC3 0x83 metafied).
	data := []byte{':', ' ', '1', '7', '0', '0', '0', '0', '0', '0', '0', '0', ':', '0', ';',
		'e', 'c', 'h', 'o', ' ', 0xC3, 0x83, 0xA3, '\n'}
	got := ParseZsh(data)
	if len(got) != 1 || got[0].Cmd != "echo Ã" {
		t.Errorf("got %+v", got)
	}
}

func TestParseBashWithTimestamps(t *testing.T) {
	data := []byte("#1700000000\ngit status\n#1700000005\nmake test\nno-timestamp\n")
	got := ParseBash(data)
	want := []Entry{
		{When: time.Unix(1700000000, 0), Cmd: "git status"},
		{When: time.Unix(1700000005, 0), Cmd: "make test"},
		{Cmd: "no-timestamp"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseBashKeepsComments(t *testing.T) {
	// A short "#..." line is a comment typed by the user, not a timestamp.
	got := ParseBash([]byte("#hello\nls\n"))
	want := []Entry{{Cmd: "#hello"}, {Cmd: "ls"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestFormatBashFlattensMultiline(t *testing.T) {
	got := string(FormatBash([]Entry{{Cmd: "for f in *; do\necho $f\ndone"}}))
	want := "for f in *; do; echo $f; done\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDetect(t *testing.T) {
	if got := Detect([]byte(": 1700000000:0;ls\n")); got != ShellZsh {
		t.Errorf("extended format detected as %q", got)
	}
	if got := Detect([]byte("ls\ncd /tmp\n")); got != ShellBash {
		t.Errorf("plain lines detected as %q", got)
	}
	if got := Detect([]byte{'l', 's', ' ', 0xC3, 0x83, 0xA3, '\n'}); got != ShellZsh {
		t.Errorf("metafied bytes detected as %q", got)
	}
}

func TestLastNViaSlice(t *testing.T) {
	data := []byte("one\ntwo\nthree\nfour\n")
	entries := ParseBash(data)
	if len(entries) != 4 {
		t.Fatalf("parsed %d entries", len(entries))
	}
	last2 := entries[len(entries)-2:]
	if last2[0].Cmd != "three" || last2[1].Cmd != "four" {
		t.Errorf("last two = %+v", last2)
	}
}
