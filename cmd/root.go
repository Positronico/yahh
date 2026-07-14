// Package cmd implements the yahh CLI.
package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Positronico/yahh/internal/histfmt"
	"github.com/Positronico/yahh/internal/paths"
	"github.com/Positronico/yahh/internal/registry"
)

// Set at build time via -ldflags (see .goreleaser.yaml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var dataDirFlag string

var rootCmd = &cobra.Command{
	Use:   "yahh",
	Short: "Per-project shell history realms for zsh and bash",
	Long: `yahh (Yet Another History Hack) keeps a separate shell command history
per project. Register a directory as a "realm" and every shell session
inside that directory tree records to — and recalls from — the realm's
own history file; leave the tree and your global history returns.

Activate it by adding to your shell rc file (or run: yahh install):

  eval "$(yahh init zsh)"    # ~/.zshrc
  eval "$(yahh init bash)"   # ~/.bashrc`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dataDirFlag, "data-dir", "",
		"override the data directory (default $YAHH_DATA_DIR or ~/.local/share/yahh)")
}

// exitError carries a specific process exit code; an empty message exits silently.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func exitCode(code int) error { return &exitError{code: code} }

// Execute runs the CLI and returns the process exit code.
func Execute() int {
	err := rootCmd.Execute()
	if err == nil {
		return 0
	}
	var ee *exitError
	if errors.As(err, &ee) {
		if ee.msg != "" {
			fmt.Fprintln(os.Stderr, "yahh: "+ee.msg)
		}
		return ee.code
	}
	fmt.Fprintln(os.Stderr, "yahh: "+err.Error())
	return 1
}

func dataDir() string {
	if dataDirFlag != "" {
		return dataDirFlag
	}
	return paths.DataDir()
}

func openDB() (*registry.DB, error) {
	return registry.Open(dataDir())
}

// realmShells lists the per-realm history file variants.
var realmShells = []string{histfmt.ShellZsh, histfmt.ShellBash}

// ensureHistFile creates a history file mode 0600 (histories can hold
// secrets) if it does not exist yet.
func ensureHistFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}

// currentShell guesses the user's shell from $SHELL (defaults to zsh).
func currentShell() string {
	if strings.Contains(filepath.Base(os.Getenv("SHELL")), "bash") {
		return histfmt.ShellBash
	}
	return histfmt.ShellZsh
}

// defaultGlobalHist returns the conventional global history file for a shell.
func defaultGlobalHist(shell string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	if shell == histfmt.ShellBash {
		return filepath.Join(home, ".bash_history")
	}
	return filepath.Join(home, ".zsh_history")
}

// confirm prompts on stderr and reads a y/N answer from stdin.
func confirm(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(line)
	return strings.EqualFold(line, "y") || strings.EqualFold(line, "yes")
}

// lookupRealm finds a realm by name first, then by normalized path.
func lookupRealm(db *registry.DB, arg string) (registry.Realm, error) {
	if r, ok, err := db.ByName(arg); err != nil {
		return registry.Realm{}, err
	} else if ok {
		return r, nil
	}
	if norm, err := paths.NormalizeDir(arg); err == nil {
		if r, ok, err := db.ByPath(norm); err == nil && ok {
			return r, nil
		}
	}
	return registry.Realm{}, fmt.Errorf("no realm named or registered at %q (see `yahh list`)", arg)
}

var ageRe = regexp.MustCompile(`^(\d+)([dw])$`)

// parseAge parses durations, adding d(ays) and w(eeks) suffixes to what
// time.ParseDuration accepts.
func parseAge(s string) (time.Duration, error) {
	if m := ageRe.FindStringSubmatch(s); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, err
		}
		unit := 24 * time.Hour
		if m[2] == "w" {
			unit = 7 * 24 * time.Hour
		}
		return time.Duration(n) * unit, nil
	}
	return time.ParseDuration(s)
}

// relTime renders a timestamp as a compact age for tables.
func relTime(t, now time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", d/time.Minute)
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", d/time.Hour)
	case d < 60*24*time.Hour:
		return fmt.Sprintf("%dd ago", d/(24*time.Hour))
	default:
		return t.Format("2006-01-02")
	}
}
