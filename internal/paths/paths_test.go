package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeDirResolvesSymlinks(t *testing.T) {
	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	link := filepath.Join(tmp, "link")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	got, err := NormalizeDir(link)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(real)
	if got != want {
		t.Errorf("NormalizeDir(%q) = %q, want %q", link, got, want)
	}
}

func TestNormalizeDirTrailingSlash(t *testing.T) {
	tmp := t.TempDir()
	got, err := NormalizeDir(tmp + string(filepath.Separator))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(tmp)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeDirVanishedPath(t *testing.T) {
	tmp := t.TempDir()
	gone := filepath.Join(tmp, "gone", "deeper")
	got, err := NormalizeDir(gone)
	if err != nil {
		t.Fatal(err)
	}
	resolvedTmp, _ := filepath.EvalSymlinks(tmp)
	want := filepath.Join(resolvedTmp, "gone", "deeper")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeDirVanishedUnderSymlinkedAncestor(t *testing.T) {
	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	link := filepath.Join(tmp, "link")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	got, err := NormalizeDir(filepath.Join(link, "missing"))
	if err != nil {
		t.Fatal(err)
	}
	resolvedReal, _ := filepath.EvalSymlinks(real)
	want := filepath.Join(resolvedReal, "missing")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"myproj":              "myproj",
		"My Project (2024)!!": "My-Project-2024-",
		"...hidden":           "hidden",
		"--flags":             "flags",
		"":                    "realm",
		"ünïcode":             "n-code",
	}
	for in, want := range cases {
		if got := SanitizeName(in); got != want {
			t.Errorf("SanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
	long := SanitizeName(string(make([]byte, 100)))
	if len(long) > 40 {
		t.Errorf("SanitizeName did not truncate: %d chars", len(long))
	}
}
