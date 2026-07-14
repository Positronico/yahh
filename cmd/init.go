package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Positronico/yahh/internal/shellhook"
)

var (
	initNoAutoClean  bool
	initNoCompletion bool
)

var initCmd = &cobra.Command{
	Use:       "init <zsh|bash>",
	Short:     "Print the shell integration script (eval it from your rc file)",
	Long:      `Prints the shell hook script. Add to your rc file:` + "\n\n  eval \"$(yahh init zsh)\"    # ~/.zshrc\n  eval \"$(yahh init bash)\"   # ~/.bashrc",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"zsh", "bash"},
	RunE: func(cmd *cobra.Command, args []string) error {
		bin, err := os.Executable()
		if err != nil || bin == "" {
			bin = "yahh"
		}
		script, err := shellhook.Render(args[0], shellhook.Params{
			BinPath:     bin,
			DataDir:     dataDir(),
			AutoClean:   !initNoAutoClean,
			Completions: !initNoCompletion,
		})
		if err != nil {
			return err
		}
		fmt.Print(script)
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initNoAutoClean, "no-autoclean", false, "do not trigger the throttled background clean at shell startup")
	initCmd.Flags().BoolVar(&initNoCompletion, "no-completions", false, "do not source shell completions")
	rootCmd.AddCommand(initCmd)
}
