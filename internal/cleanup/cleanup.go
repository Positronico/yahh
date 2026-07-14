// Package cleanup detects and removes realms whose directories are gone.
package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Positronico/yahh/internal/registry"
)

// Report summarizes a Scan.
type Report struct {
	Marked  []registry.Realm // directory missing, newly flagged as orphaned
	Healed  []registry.Realm // directory reappeared, flag cleared
	Expired []registry.Realm // orphaned longer than the grace period
}

// Empty reports whether the scan found nothing to act on.
func (r Report) Empty() bool {
	return len(r.Marked) == 0 && len(r.Healed) == 0 && len(r.Expired) == 0
}

// Scan checks every realm's directory. Missing directories are flagged
// (never deleted on first sighting); directories that reappear are
// unflagged; realms orphaned longer than grace are returned as Expired
// for the caller to remove. With apply=false nothing is written.
func Scan(db *registry.DB, now time.Time, grace time.Duration, apply bool) (Report, error) {
	var rep Report
	realms, err := db.All()
	if err != nil {
		return rep, err
	}
	for _, r := range realms {
		_, statErr := os.Stat(r.Path)
		exists := statErr == nil
		switch {
		case exists && r.Orphaned():
			rep.Healed = append(rep.Healed, r)
			if apply {
				if err := db.ClearOrphaned(r.ID); err != nil {
					return rep, err
				}
			}
		case !exists && !r.Orphaned():
			rep.Marked = append(rep.Marked, r)
			if apply {
				if err := db.MarkOrphaned(r.ID, now); err != nil {
					return rep, err
				}
			}
		case !exists && now.Sub(r.OrphanedAt) >= grace:
			rep.Expired = append(rep.Expired, r)
		}
	}
	return rep, nil
}

// Remove deletes a realm from the registry. Its history files are moved
// to the archive directory (or deleted outright with purge). Returns the
// archived file paths.
func Remove(db *registry.DB, r registry.Realm, purge bool, now time.Time) ([]string, error) {
	var archived []string
	for _, shell := range []string{"zsh", "bash"} {
		src := db.HistFile(r, shell)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if purge {
			if err := os.Remove(src); err != nil {
				return archived, err
			}
			continue
		}
		archDir := filepath.Join(db.DataDir(), "archive")
		if err := os.MkdirAll(archDir, 0o700); err != nil {
			return archived, err
		}
		dst := filepath.Join(archDir, fmt.Sprintf("%s.removed.%s",
			filepath.Base(src), now.Format("20060102T150405")))
		if err := os.Rename(src, dst); err != nil {
			return archived, err
		}
		archived = append(archived, dst)
	}
	return archived, db.Delete(r.ID)
}

// PurgeArchive deletes archived history files older than olderThan.
// With apply=false it only reports what would be deleted.
func PurgeArchive(dataDir string, olderThan time.Duration, now time.Time, apply bool) ([]string, error) {
	archDir := filepath.Join(dataDir, "archive")
	dirEntries, err := os.ReadDir(archDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var purged []string
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		info, err := de.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) < olderThan {
			continue
		}
		p := filepath.Join(archDir, de.Name())
		if apply {
			if err := os.Remove(p); err != nil {
				return purged, err
			}
		}
		purged = append(purged, p)
	}
	return purged, nil
}
