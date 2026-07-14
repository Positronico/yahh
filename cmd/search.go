package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Positronico/yahh/internal/histfmt"
)

var (
	searchRealm  string
	searchRegex  bool
	searchGlobal bool
	searchJSON   bool
)

type searchHit struct {
	Realm string     `json:"realm"`
	When  *time.Time `json:"when,omitempty"`
	Cmd   string     `json:"cmd"`
}

var searchCmd = &cobra.Command{
	Use:   "search <term>",
	Short: "Search across all realm histories",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var match func(string) bool
		if searchRegex {
			re, err := regexp.Compile(args[0])
			if err != nil {
				return err
			}
			match = re.MatchString
		} else {
			needle := strings.ToLower(args[0])
			match = func(s string) bool { return strings.Contains(strings.ToLower(s), needle) }
		}

		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()
		realms, err := db.All()
		if err != nil {
			return err
		}
		if searchRealm != "" {
			r, err := lookupRealm(db, searchRealm)
			if err != nil {
				return err
			}
			realms = realms[:0]
			realms = append(realms, r)
		}

		var hits []searchHit
		// A realm's zsh and bash files can hold identical seeded entries;
		// report each (realm, time, command) once.
		seen := make(map[string]bool)
		add := func(label string, entries []histfmt.Entry) {
			for _, e := range entries {
				if !match(e.Cmd) {
					continue
				}
				key := fmt.Sprintf("%s\x00%d\x00%s", label, e.When.Unix(), e.Cmd)
				if seen[key] {
					continue
				}
				seen[key] = true
				h := searchHit{Realm: label, Cmd: e.Cmd}
				if !e.When.IsZero() {
					t := e.When
					h.When = &t
				}
				hits = append(hits, h)
			}
		}
		for _, r := range realms {
			for _, shell := range realmShells {
				entries, err := histfmt.ParseFile(db.HistFile(r, shell), shell)
				if err != nil {
					return err
				}
				add(r.Name, entries)
			}
		}
		if searchGlobal {
			for _, shell := range realmShells {
				data, err := os.ReadFile(defaultGlobalHist(shell))
				if err != nil {
					continue
				}
				add("global", histfmt.Parse(data, histfmt.Detect(data)))
			}
		}

		if searchJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(hits); err != nil {
				return err
			}
			if len(hits) == 0 {
				return exitCode(1)
			}
			return nil
		}
		if len(hits) == 0 {
			fmt.Fprintln(os.Stderr, "no matches")
			return exitCode(1)
		}
		for _, h := range hits {
			when := "                "
			if h.When != nil {
				when = h.When.Format("2006-01-02 15:04")
			}
			fmt.Printf("[%s]  %s  %s\n", h.Realm, when, h.Cmd)
		}
		return nil
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchRealm, "realm", "", "search only this realm")
	searchCmd.Flags().BoolVar(&searchRegex, "regex", false, "treat the term as a regular expression")
	searchCmd.Flags().BoolVar(&searchGlobal, "global", false, "also search your global history files")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output JSON")
	rootCmd.AddCommand(searchCmd)
}
