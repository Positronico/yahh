package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Positronico/yahh/internal/paths"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show yahh's state and the realm covering the current directory",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()
		enabled, err := db.Enabled()
		if err != nil {
			return err
		}
		realms, err := db.All()
		if err != nil {
			return err
		}
		lastClean, err := db.LastCleanAt()
		if err != nil {
			return err
		}

		fmt.Printf("Enabled:    %v\n", enabled)
		fmt.Printf("Data dir:   %s\n", db.DataDir())
		fmt.Printf("Realms:     %d\n", len(realms))
		if lastClean.IsZero() {
			fmt.Printf("Last clean: never\n")
		} else {
			fmt.Printf("Last clean: %s\n", lastClean.Format("2006-01-02 15:04"))
		}

		cwd, err := os.Getwd()
		if err != nil {
			return nil
		}
		norm, err := paths.NormalizeDir(cwd)
		if err != nil {
			return nil
		}
		if r, ok, err := db.Resolve(norm); err == nil && ok {
			fmt.Printf("Here:       realm %q (root %s)\n", r.Name, r.Path)
		} else {
			fmt.Printf("Here:       no realm (global history)\n")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
