package shellhook

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderZsh(t *testing.T) {
	out, err := Render("zsh", Params{BinPath: "/opt/bin/yahh", DataDir: "/data/yahh", AutoClean: true, Completions: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"_yahh_bin='/opt/bin/yahh'",
		"_yahh_snap='/data/yahh/snapshot'",
		"_yahh_lookup",
		"yahh1 enabled=",
		"add-zsh-hook chpwd _yahh_chpwd",
		"add-zsh-hook precmd _yahh_startup",
		"clean --auto",
		"completion zsh",
		"fc -p",
		"fc -AI",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("zsh script missing %q", want)
		}
	}
	// v1's bug: a bare `precmd()` assignment clobbers other plugins.
	if strings.Contains(out, "precmd()") {
		t.Error("zsh script defines precmd() directly")
	}
}

func TestRenderBash(t *testing.T) {
	out, err := Render("bash", Params{BinPath: "/opt/bin/yahh", DataDir: "/data/yahh", AutoClean: true, Completions: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"_yahh_bin='/opt/bin/yahh'",
		"_yahh_snap='/data/yahh/snapshot'",
		"_yahh_lookup",
		"yahh1 enabled=",
		"PROMPT_COMMAND",
		"history -a",
		"clean --auto",
		"completion bash",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("bash script missing %q", want)
		}
	}
	saveIdx := strings.Index(out, "history -a")
	assignIdx := strings.Index(out, "HISTFILE=\"$histfile\"")
	if saveIdx < 0 || assignIdx < 0 || saveIdx > assignIdx {
		t.Error("the history save must run before HISTFILE is reassigned")
	}
	if !strings.Contains(out, "history -w") {
		t.Error("missing the bash 3.2 history -w fallback")
	}
}

func TestRenderFlagsOff(t *testing.T) {
	out, err := Render("zsh", Params{BinPath: "yahh", DataDir: "/data"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "clean --auto") || strings.Contains(out, "completion zsh") {
		t.Error("optional blocks rendered despite being disabled")
	}
}

func TestRenderUnsupportedShell(t *testing.T) {
	if _, err := Render("fish", Params{BinPath: "yahh", DataDir: "/data"}); err == nil {
		t.Error("expected error for unsupported shell")
	}
}

func TestQuote(t *testing.T) {
	if got := Quote("/path/with 'quote'"); got != `'/path/with '\''quote'\'''` {
		t.Errorf("Quote = %s", got)
	}
}

// The rendered scripts must parse in their target shells.
func TestScriptSyntax(t *testing.T) {
	for _, shell := range []string{"zsh", "bash"} {
		bin, err := exec.LookPath(shell)
		if err != nil {
			t.Logf("%s not installed; skipping syntax check", shell)
			continue
		}
		out, err := Render(shell, Params{BinPath: "/opt/bin/yahh", DataDir: "/data/yahh", AutoClean: true, Completions: true})
		if err != nil {
			t.Fatal(err)
		}
		script := filepath.Join(t.TempDir(), "hook."+shell)
		if err := os.WriteFile(script, []byte(out), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(bin, "-n", script)
		if msg, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s -n failed: %v\n%s", shell, err, msg)
		}
	}
}
