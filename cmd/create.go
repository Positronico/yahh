package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/Positronico/yahh/internal/histfmt"
	"github.com/Positronico/yahh/internal/paths"
	"github.com/Positronico/yahh/internal/registry"
)

var (
	createName   string
	createImport string
	createFrom   string
	createForce  bool
)

var createCmd = &cobra.Command{
	Use:   "create [dir]",
	Short: "Register a directory (and its subtree) as a history realm",
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
		info, err := os.Stat(norm)
		if err != nil {
			return fmt.Errorf("cannot create a realm at %s: %w", norm, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", norm)
		}
		if norm == "/" {
			return errors.New("refusing to create a realm covering the entire filesystem")
		}
		if home, herr := os.UserHomeDir(); herr == nil && !createForce {
			if normHome, herr := paths.NormalizeDir(home); herr == nil && norm == normHome {
				return errors.New("refusing to cover your entire home directory (use --force to override)")
			}
		}

		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		if existing, ok, err := db.ByPath(norm); err != nil {
			return err
		} else if ok {
			return fmt.Errorf("%s is already registered as realm %q", norm, existing.Name)
		}

		name := createName
		if name == "" {
			name = filepath.Base(norm)
		}
		r, err := db.Create(name, norm, time.Now())
		if err != nil {
			return err
		}
		for _, shell := range realmShells {
			if err := ensureHistFile(db.HistFile(r, shell)); err != nil {
				return err
			}
		}

		seeded := 0
		if createImport != "" || createFrom != "" {
			n := 1000
			if createImport != "" {
				if n, err = strconv.Atoi(createImport); err != nil || n <= 0 {
					return fmt.Errorf("invalid --import count %q", createImport)
				}
			}
			if seeded, err = seedRealm(db, r, createFrom, n); err != nil {
				return err
			}
		}

		fmt.Printf("Created realm %q covering %s\n", r.Name, r.Path)
		if seeded > 0 {
			fmt.Printf("Seeded %d entries from %s\n", seeded, seedSource(createFrom))
		}
		if enabled, _ := db.Enabled(); !enabled {
			fmt.Println("Note: yahh is currently disabled; run `yahh enable` to activate realm switching")
		}
		return nil
	},
}

func seedSource(from string) string {
	if from != "" {
		return from
	}
	return defaultGlobalHist(currentShell())
}

// seedRealm copies the last n entries of the source history into both of
// the realm's per-shell history files (converting formats as needed).
func seedRealm(db *registry.DB, r registry.Realm, from string, n int) (int, error) {
	data, err := os.ReadFile(seedSource(from))
	if err != nil {
		return 0, fmt.Errorf("cannot read import source: %w", err)
	}
	entries := histfmt.Parse(data, histfmt.Detect(data))
	if len(entries) > n {
		entries = entries[len(entries)-n:]
	}
	if len(entries) == 0 {
		return 0, nil
	}
	for _, shell := range realmShells {
		if err := histfmt.AppendFile(db.HistFile(r, shell), shell, entries); err != nil {
			return 0, err
		}
	}
	return len(entries), nil
}

func init() {
	createCmd.Flags().StringVar(&createName, "name", "", "realm name (default: directory basename)")
	createCmd.Flags().StringVar(&createImport, "import", "", "seed the realm with the last N global history entries")
	createCmd.Flags().Lookup("import").NoOptDefVal = "1000"
	createCmd.Flags().StringVar(&createFrom, "from", "", "history file to import from (default: your shell's global history)")
	createCmd.Flags().BoolVar(&createForce, "force", false, "allow registering your home directory")
	rootCmd.AddCommand(createCmd)
}
