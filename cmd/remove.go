package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/Positronico/yahh/internal/cleanup"
	"github.com/Positronico/yahh/internal/histfmt"
	"github.com/Positronico/yahh/internal/paths"
	"github.com/Positronico/yahh/internal/registry"
)

var (
	removeName  string
	removeMerge bool
	removeInto  string
	removePurge bool
	removeYes   bool
)

var removeCmd = &cobra.Command{
	Use:   "remove [dir]",
	Short: "Unregister a realm; its history is archived (or merged/purged)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		var r registry.Realm
		if removeName != "" {
			var ok bool
			if r, ok, err = db.ByName(removeName); err != nil {
				return err
			} else if !ok {
				return fmt.Errorf("no realm named %q", removeName)
			}
		} else {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}
			norm, err := paths.NormalizeDir(target)
			if err != nil {
				return err
			}
			var ok bool
			if r, ok, err = db.Resolve(norm); err != nil {
				return err
			} else if !ok {
				return fmt.Errorf("no realm covers %s", norm)
			}
		}

		if !removeYes {
			verb := "archived"
			if removePurge {
				verb = "deleted"
			}
			if removeMerge {
				verb = "merged into your global history, then " + verb
			}
			prompt := fmt.Sprintf("Remove realm %q (%s; %d history entries will be %s)?",
				r.Name, r.Path, countRealmEntries(db, r), verb)
			if !confirm(prompt) {
				return errors.New("cancelled")
			}
		}

		if removeMerge {
			if err := mergeRealm(db, r, removeInto); err != nil {
				return err
			}
		}
		archived, err := cleanup.Remove(db, r, removePurge, time.Now())
		if err != nil {
			return err
		}

		fmt.Printf("Removed realm %q (%s)\n", r.Name, r.Path)
		for _, f := range archived {
			fmt.Printf("Archived history: %s\n", f)
		}
		fmt.Println("Shells currently inside this realm return to the global history on their next directory change.")
		return nil
	},
}

func countRealmEntries(db *registry.DB, r registry.Realm) int {
	total := 0
	for _, shell := range realmShells {
		entries, _ := histfmt.ParseFile(db.HistFile(r, shell), shell)
		total += len(entries)
	}
	return total
}

// mergeRealm appends the realm's history into the global history file(s),
// skipping commands the destination already contains.
func mergeRealm(db *registry.DB, r registry.Realm, into string) error {
	if into != "" {
		var all []histfmt.Entry
		for _, shell := range realmShells {
			entries, err := histfmt.ParseFile(db.HistFile(r, shell), shell)
			if err != nil {
				return err
			}
			all = append(all, entries...)
		}
		sort.SliceStable(all, func(i, j int) bool { return all[i].When.Before(all[j].When) })
		destData, _ := os.ReadFile(into)
		destShell := histfmt.Detect(destData)
		if len(destData) == 0 {
			destShell = currentShell()
		}
		return histfmt.AppendFile(into, destShell, dedupeAgainst(all, destData, destShell))
	}
	for _, shell := range realmShells {
		entries, err := histfmt.ParseFile(db.HistFile(r, shell), shell)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			continue
		}
		dest := defaultGlobalHist(shell)
		destData, _ := os.ReadFile(dest)
		if err := histfmt.AppendFile(dest, shell, dedupeAgainst(entries, destData, shell)); err != nil {
			return err
		}
	}
	return nil
}

func dedupeAgainst(entries []histfmt.Entry, destData []byte, destShell string) []histfmt.Entry {
	seen := make(map[string]bool)
	for _, e := range histfmt.Parse(destData, destShell) {
		seen[e.Cmd] = true
	}
	var out []histfmt.Entry
	for _, e := range entries {
		if !seen[e.Cmd] {
			out = append(out, e)
			seen[e.Cmd] = true
		}
	}
	return out
}

func init() {
	removeCmd.Flags().StringVar(&removeName, "name", "", "remove the realm with this name instead of the one covering [dir]")
	removeCmd.Flags().BoolVar(&removeMerge, "merge", false, "fold the realm's history back into your global history first")
	removeCmd.Flags().StringVar(&removeInto, "into", "", "merge destination file (implies --merge semantics for that file)")
	removeCmd.Flags().BoolVar(&removePurge, "purge", false, "delete history files instead of archiving them")
	removeCmd.Flags().BoolVarP(&removeYes, "yes", "y", false, "do not ask for confirmation")
	rootCmd.AddCommand(removeCmd)
}
