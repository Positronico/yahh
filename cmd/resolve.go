package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Positronico/yahh/internal/histfmt"
	"github.com/Positronico/yahh/internal/paths"
)

var (
	resolveShell string
	resolvePwd   string
)

// resolve is the hook hot path: it must be fast and silent. Exit codes:
// 0 = realm found (prints "<root>\t<histfile>"), 1 = no realm, 2 = disabled.
var resolveCmd = &cobra.Command{
	Use:    "resolve --shell <zsh|bash> --pwd <dir>",
	Short:  "Print the realm root and history file for a directory (hook plumbing)",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if resolveShell != histfmt.ShellZsh && resolveShell != histfmt.ShellBash {
			return exitCode(1)
		}
		db, err := openDB()
		if err != nil {
			return exitCode(1)
		}
		defer db.Close()
		// The hooks call resolve when the snapshot is unavailable; heal it
		// so subsequent directory changes stay in pure shell.
		_ = db.EnsureSnapshot()
		enabled, err := db.Enabled()
		if err != nil {
			return exitCode(1)
		}
		if !enabled {
			return exitCode(2)
		}
		norm, err := paths.NormalizeDir(resolvePwd)
		if err != nil {
			return exitCode(1)
		}
		r, ok, err := db.Resolve(norm)
		if err != nil || !ok {
			return exitCode(1)
		}
		histfile := db.HistFile(r, resolveShell)
		if err := ensureHistFile(histfile); err != nil {
			return exitCode(1)
		}
		_ = db.TouchLastUsed(r.ID, time.Now(), time.Hour)
		fmt.Printf("%s\t%s\n", r.Path, histfile)
		return nil
	},
}

func init() {
	resolveCmd.Flags().StringVar(&resolveShell, "shell", "", "shell whose history file to resolve (zsh|bash)")
	resolveCmd.Flags().StringVar(&resolvePwd, "pwd", "", "directory to resolve")
	_ = resolveCmd.MarkFlagRequired("shell")
	_ = resolveCmd.MarkFlagRequired("pwd")
	rootCmd.AddCommand(resolveCmd)
}
