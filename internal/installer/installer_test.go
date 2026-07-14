package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallIsIdempotent(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".zshrc")
	if err := os.WriteFile(rc, []byte("export PATH=$PATH:/opt/bin\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := Install(rc, "zsh")
	if err != nil || !changed {
		t.Fatalf("first install = (%v, %v)", changed, err)
	}
	changed, err = Install(rc, "zsh")
	if err != nil || changed {
		t.Fatalf("second install = (%v, %v), want (false, nil)", changed, err)
	}
	content, _ := os.ReadFile(rc)
	if strings.Count(string(content), BeginMarker) != 1 {
		t.Errorf("marker appears %d times", strings.Count(string(content), BeginMarker))
	}
	if !strings.Contains(string(content), `eval "$(yahh init zsh)"`) {
		t.Errorf("missing eval line:\n%s", content)
	}
}

func TestInstallCreatesMissingFile(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".bashrc")
	changed, err := Install(rc, "bash")
	if err != nil || !changed {
		t.Fatalf("install = (%v, %v)", changed, err)
	}
	content, _ := os.ReadFile(rc)
	if !strings.Contains(string(content), "yahh init bash") {
		t.Errorf("content:\n%s", content)
	}
}

func TestUninstallRemovesOnlyTheBlock(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".zshrc")
	before := "alias ll='ls -la'\n"
	after := "export EDITOR=vim\n"
	if err := os.WriteFile(rc, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(rc, "zsh"); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(rc, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(after)
	f.Close()

	removed, err := Uninstall(rc)
	if err != nil || !removed {
		t.Fatalf("uninstall = (%v, %v)", removed, err)
	}
	content, _ := os.ReadFile(rc)
	if strings.Contains(string(content), "yahh") {
		t.Errorf("yahh content left behind:\n%s", content)
	}
	if !strings.Contains(string(content), before) || !strings.Contains(string(content), after) {
		t.Errorf("surrounding content damaged:\n%s", content)
	}
}

func TestUninstallNoBlock(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".zshrc")
	os.WriteFile(rc, []byte("plain\n"), 0o644)
	removed, err := Uninstall(rc)
	if err != nil || removed {
		t.Fatalf("uninstall = (%v, %v), want (false, nil)", removed, err)
	}
	removed, err = Uninstall(filepath.Join(t.TempDir(), "missing"))
	if err != nil || removed {
		t.Fatalf("uninstall missing file = (%v, %v), want (false, nil)", removed, err)
	}
}
