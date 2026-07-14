package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Positronico/yahh/internal/installer"
)

var (
	installShells []string
	installRC     string
	uninstallRC   string
	uninstallPurge bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Add the yahh integration to your shell rc file(s)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		shells := installShells
		if len(shells) == 0 {
			shells = detectShells()
		}
		if len(shells) == 0 {
			return errors.New("could not detect zsh or bash; pass --shell zsh and/or --shell bash")
		}
		if installRC != "" && len(shells) != 1 {
			return errors.New("--rc requires exactly one --shell")
		}
		changed := false
		for _, sh := range shells {
			if sh != "zsh" && sh != "bash" {
				return fmt.Errorf("unsupported shell %q (supported: zsh, bash)", sh)
			}
			rc := installRC
			if rc == "" {
				rc = rcFileFor(sh)
			}
			did, err := installer.Install(rc, sh)
			if err != nil {
				return err
			}
			if did {
				fmt.Printf("Added the yahh init block for %s to %s\n", sh, rc)
				changed = true
			} else {
				fmt.Printf("%s already contains the yahh init block\n", rc)
			}
		}
		if changed {
			fmt.Println("Restart your shell (or run `exec $SHELL`) to activate.")
		}
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the yahh integration from your shell rc file(s)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var rcs []string
		if uninstallRC != "" {
			rcs = []string{uninstallRC}
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			rcs = []string{rcFileFor("zsh"), filepath.Join(home, ".bashrc"), filepath.Join(home, ".bash_profile")}
		}
		removedAny := false
		for _, rc := range rcs {
			removed, err := installer.Uninstall(rc)
			if err != nil {
				return err
			}
			if removed {
				fmt.Printf("Removed the yahh init block from %s\n", rc)
				removedAny = true
			}
		}
		if !removedAny {
			fmt.Println("No yahh init block found in your rc files.")
		}
		if uninstallPurge {
			dd := dataDir()
			if confirm(fmt.Sprintf("Delete ALL yahh data in %s (registry and every realm history)?", dd)) {
				if err := os.RemoveAll(dd); err != nil {
					return err
				}
				fmt.Printf("Deleted %s\n", dd)
			} else {
				fmt.Println("Kept data directory.")
			}
		}
		return nil
	},
}

func detectShells() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	shellEnv := filepath.Base(os.Getenv("SHELL"))
	var shells []string
	if strings.Contains(shellEnv, "zsh") || fileExists(filepath.Join(home, ".zshrc")) {
		shells = append(shells, "zsh")
	}
	if strings.Contains(shellEnv, "bash") ||
		fileExists(filepath.Join(home, ".bashrc")) || fileExists(filepath.Join(home, ".bash_profile")) {
		shells = append(shells, "bash")
	}
	return shells
}

func rcFileFor(shell string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	if shell == "zsh" {
		if zdot := os.Getenv("ZDOTDIR"); zdot != "" {
			return filepath.Join(zdot, ".zshrc")
		}
		return filepath.Join(home, ".zshrc")
	}
	// macOS terminals start login shells that read ~/.bash_profile and may
	// never read ~/.bashrc — prefer it unless it already hands off to .bashrc.
	if runtime.GOOS == "darwin" {
		profile := filepath.Join(home, ".bash_profile")
		if data, err := os.ReadFile(profile); err == nil && !strings.Contains(string(data), ".bashrc") {
			return profile
		}
	}
	return filepath.Join(home, ".bashrc")
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func init() {
	installCmd.Flags().StringSliceVar(&installShells, "shell", nil, "shell(s) to install for (zsh, bash; default: autodetect)")
	installCmd.Flags().StringVar(&installRC, "rc", "", "rc file to modify (requires exactly one --shell)")
	uninstallCmd.Flags().StringVar(&uninstallRC, "rc", "", "rc file to modify")
	uninstallCmd.Flags().BoolVar(&uninstallPurge, "purge", false, "also delete the data directory (registry and histories)")
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}
