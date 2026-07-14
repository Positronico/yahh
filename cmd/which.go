package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Positronico/yahh/internal/paths"
)

var whichCmd = &cobra.Command{
	Use:   "which [dir]",
	Short: "Show the realm covering a directory (default: current directory)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := "."
		if len(args) == 1 {
			target = args[0]
		}
		norm, err := paths.NormalizeDir(target)
		if err != nil {
			return err
		}
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()
		r, ok, err := db.Resolve(norm)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Printf("No realm covers %s (global history applies)\n", norm)
			return exitCode(1)
		}
		fmt.Printf("Realm:   %s\nRoot:    %s\nHistory: %s\n         %s\n",
			r.Name, r.Path, db.HistFile(r, "zsh"), db.HistFile(r, "bash"))
		if enabled, _ := db.Enabled(); !enabled {
			fmt.Println("Note: yahh is disabled (`yahh enable` to activate)")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(whichCmd)
}
