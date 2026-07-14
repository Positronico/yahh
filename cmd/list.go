package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var (
	listJSON  bool
	listPaths bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all realms",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()
		realms, err := db.All()
		if err != nil {
			return err
		}

		if listPaths {
			for _, r := range realms {
				fmt.Println(r.Path)
			}
			return nil
		}

		if listJSON {
			type realmJSON struct {
				Name       string     `json:"name"`
				Path       string     `json:"path"`
				Entries    int        `json:"entries"`
				CreatedAt  time.Time  `json:"created_at"`
				LastUsedAt *time.Time `json:"last_used_at,omitempty"`
				OrphanedAt *time.Time `json:"orphaned_at,omitempty"`
			}
			out := make([]realmJSON, 0, len(realms))
			for _, r := range realms {
				j := realmJSON{Name: r.Name, Path: r.Path, Entries: countRealmEntries(db, r), CreatedAt: r.CreatedAt}
				if !r.LastUsedAt.IsZero() {
					t := r.LastUsedAt
					j.LastUsedAt = &t
				}
				if r.Orphaned() {
					t := r.OrphanedAt
					j.OrphanedAt = &t
				}
				out = append(out, j)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		if len(realms) == 0 {
			fmt.Println("No realms yet. Run `yahh create` inside a project directory.")
			return nil
		}
		now := time.Now()
		w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPATH\tENTRIES\tLAST USED\tSTATE")
		for _, r := range realms {
			state := "ok"
			if r.Orphaned() {
				state = "orphaned " + relTime(r.OrphanedAt, now)
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				r.Name, r.Path, countRealmEntries(db, r), relTime(r.LastUsedAt, now), state)
		}
		return w.Flush()
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output JSON")
	listCmd.Flags().BoolVar(&listPaths, "paths", false, "output only realm paths (for scripting)")
	rootCmd.AddCommand(listCmd)
}
