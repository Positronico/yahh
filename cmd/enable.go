package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable realm switching (all shells, persisted)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()
		if err := db.SetEnabled(true); err != nil {
			return err
		}
		fmt.Println("yahh enabled. Each shell applies it on its next directory change (`cd .` to apply now).")
		return nil
	},
}

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable realm switching (all shells, persisted)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()
		if err := db.SetEnabled(false); err != nil {
			return err
		}
		fmt.Println("yahh disabled. Each shell returns to the global history on its next directory change (`cd .` to apply now).")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(disableCmd)
}
