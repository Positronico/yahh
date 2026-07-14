package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Positronico/yahh/internal/paths"
	"github.com/Positronico/yahh/internal/registry"
)

var mvCmd = &cobra.Command{
	Use:   "mv <realm> <new-dir>",
	Short: "Re-point a realm at a directory that moved",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()
		r, err := lookupRealm(db, args[0])
		if err != nil {
			return err
		}
		norm, err := paths.NormalizeDir(args[1])
		if err != nil {
			return err
		}
		info, err := os.Stat(norm)
		if err != nil {
			return fmt.Errorf("cannot move realm to %s: %w", norm, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", norm)
		}
		if err := db.SetPath(r.ID, norm); err != nil {
			return err
		}
		fmt.Printf("Realm %q now covers %s\n", r.Name, norm)
		return nil
	},
}

var renameCmd = &cobra.Command{
	Use:   "rename <realm> <new-name>",
	Short: "Rename a realm",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()
		r, err := lookupRealm(db, args[0])
		if err != nil {
			return err
		}
		newName := paths.SanitizeName(args[1])
		if err := db.SetName(r.ID, newName); err != nil {
			return err
		}
		// History file names derive from the realm name — move them along.
		renamed := registry.Realm{ID: r.ID, Name: newName}
		for _, shell := range realmShells {
			old := db.HistFile(r, shell)
			if _, err := os.Stat(old); err == nil {
				if err := os.Rename(old, db.HistFile(renamed, shell)); err != nil {
					return err
				}
			}
		}
		fmt.Printf("Renamed realm %q to %q\n", r.Name, newName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mvCmd)
	rootCmd.AddCommand(renameCmd)
}
