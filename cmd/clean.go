package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Positronico/yahh/internal/cleanup"
)

var (
	cleanAuto         bool
	cleanDryRun       bool
	cleanYes          bool
	cleanGrace        string
	cleanPurgeArchive string
)

// autoCleanInterval throttles the shell-startup-triggered clean.
const autoCleanInterval = 7 * 24 * time.Hour

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Flag and remove realms whose directories no longer exist",
	Long: `Checks every realm's directory. Missing directories are first flagged as
orphaned; realms that stay orphaned longer than the grace period are
removed and their history files archived. Directories that reappear
(e.g. an unmounted volume) are unflagged.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		grace, err := parseAge(cleanGrace)
		if err != nil {
			return fmt.Errorf("invalid --grace: %w", err)
		}
		db, err := openDB()
		if err != nil {
			if cleanAuto {
				return nil
			}
			return err
		}
		defer db.Close()
		now := time.Now()

		if cleanAuto {
			// Best-effort and quiet: this runs backgrounded at shell startup.
			claimed, err := db.ClaimAutoClean(now, autoCleanInterval)
			if err != nil || !claimed {
				return nil
			}
			rep, err := cleanup.Scan(db, now, grace, true)
			if err != nil {
				return nil
			}
			for _, r := range rep.Expired {
				_, _ = cleanup.Remove(db, r, false, now)
			}
			return nil
		}

		apply := !cleanDryRun
		rep, err := cleanup.Scan(db, now, grace, apply)
		if err != nil {
			return err
		}
		for _, r := range rep.Healed {
			fmt.Printf("Healed: %q — directory %s is back\n", r.Name, r.Path)
		}
		for _, r := range rep.Marked {
			fmt.Printf("Orphaned: %q — directory %s is gone (removed after %s unless it returns)\n",
				r.Name, r.Path, cleanGrace)
		}
		if len(rep.Expired) > 0 {
			fmt.Printf("Orphaned longer than %s:\n", cleanGrace)
			for _, r := range rep.Expired {
				fmt.Printf("  %q — %s (orphaned since %s)\n", r.Name, r.Path, r.OrphanedAt.Format("2006-01-02"))
			}
			switch {
			case cleanDryRun:
				fmt.Println("Dry run: these realms would be removed and their histories archived.")
			case cleanYes || confirm(fmt.Sprintf("Remove these %d realm(s) and archive their histories?", len(rep.Expired))):
				for _, r := range rep.Expired {
					archived, err := cleanup.Remove(db, r, false, now)
					if err != nil {
						return err
					}
					fmt.Printf("Removed realm %q\n", r.Name)
					for _, f := range archived {
						fmt.Printf("  archived %s\n", f)
					}
				}
			default:
				return errors.New("cancelled")
			}
		}

		purgedSomething := false
		if cmd.Flags().Changed("purge-archive") {
			age, err := parseAge(cleanPurgeArchive)
			if err != nil {
				return fmt.Errorf("invalid --purge-archive: %w", err)
			}
			candidates, err := cleanup.PurgeArchive(db.DataDir(), age, now, false)
			if err != nil {
				return err
			}
			if len(candidates) == 0 {
				fmt.Printf("No archived histories older than %s.\n", cleanPurgeArchive)
			} else {
				fmt.Printf("Archived histories older than %s:\n", cleanPurgeArchive)
				for _, f := range candidates {
					fmt.Printf("  %s\n", f)
				}
				switch {
				case cleanDryRun:
					fmt.Println("Dry run: these files would be deleted.")
				case cleanYes || confirm(fmt.Sprintf("Delete these %d archived file(s) permanently?", len(candidates))):
					if _, err := cleanup.PurgeArchive(db.DataDir(), age, now, true); err != nil {
						return err
					}
					purgedSomething = true
					fmt.Println("Archive purged.")
				default:
					return errors.New("cancelled")
				}
			}
		}

		if apply {
			_ = db.SetLastCleanAt(now)
			_ = db.Vacuum()
		}
		if rep.Empty() && !purgedSomething && !cmd.Flags().Changed("purge-archive") {
			fmt.Println("Nothing to clean: every realm's directory exists.")
		}
		return nil
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAuto, "auto", false, "throttled non-interactive mode (used by the shell hook)")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "report only; change nothing")
	cleanCmd.Flags().BoolVarP(&cleanYes, "yes", "y", false, "do not ask for confirmation")
	cleanCmd.Flags().StringVar(&cleanGrace, "grace", "30d", "how long a realm may stay orphaned before removal")
	cleanCmd.Flags().StringVar(&cleanPurgeArchive, "purge-archive", "", "also delete archived histories older than this age")
	cleanCmd.Flags().Lookup("purge-archive").NoOptDefVal = "90d"
	rootCmd.AddCommand(cleanCmd)
}
