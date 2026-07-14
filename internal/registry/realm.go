package registry

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Positronico/yahh/internal/paths"
)

// Realm is a directory tree with its own shell history.
type Realm struct {
	ID         int64
	Name       string
	Path       string
	CreatedAt  time.Time
	LastUsedAt time.Time // zero if never used
	OrphanedAt time.Time // zero while the directory exists
}

// Orphaned reports whether clean has flagged the realm's directory as missing.
func (r Realm) Orphaned() bool { return !r.OrphanedAt.IsZero() }

// ErrPathRegistered is returned when a realm already covers the exact path.
var ErrPathRegistered = errors.New("path is already registered as a realm")

// ErrNameTaken is returned by SetName when the name is in use.
var ErrNameTaken = errors.New("realm name is already taken")

const realmCols = "id, name, path, created_at, last_used_at, orphaned_at"

// Create registers a new realm. The name is sanitized and, on collision,
// suffixed with -2, -3, …; the final name is in the returned Realm.
func (d *DB) Create(name, path string, now time.Time) (Realm, error) {
	base := paths.SanitizeName(name)
	for i := 1; i <= 100; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}
		res, err := d.sql.Exec(
			"INSERT INTO realms(name, path, created_at) VALUES(?, ?, ?)",
			candidate, path, now.Unix(),
		)
		if err != nil {
			if isUniqueErr(err, "realms.path") {
				return Realm{}, ErrPathRegistered
			}
			if isUniqueErr(err, "realms.name") {
				continue
			}
			return Realm{}, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return Realm{}, err
		}
		d.dirty = true
		return Realm{ID: id, Name: candidate, Path: path, CreatedAt: time.Unix(now.Unix(), 0)}, nil
	}
	return Realm{}, fmt.Errorf("could not find a free name for %q", base)
}

// All returns every realm ordered by path.
func (d *DB) All() ([]Realm, error) {
	rows, err := d.sql.Query("SELECT " + realmCols + " FROM realms ORDER BY path")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var realms []Realm
	for rows.Next() {
		r, err := scanRealm(rows)
		if err != nil {
			return nil, err
		}
		realms = append(realms, r)
	}
	return realms, rows.Err()
}

// ByName looks a realm up by exact name.
func (d *DB) ByName(name string) (Realm, bool, error) {
	return d.one("SELECT "+realmCols+" FROM realms WHERE name = ?", name)
}

// ByPath looks a realm up by exact (normalized) path.
func (d *DB) ByPath(path string) (Realm, bool, error) {
	return d.one("SELECT "+realmCols+" FROM realms WHERE path = ?", path)
}

func (d *DB) one(query string, args ...any) (Realm, bool, error) {
	r, err := scanRealm(d.sql.QueryRow(query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return Realm{}, false, nil
	}
	if err != nil {
		return Realm{}, false, err
	}
	return r, true, nil
}

// Resolve returns the deepest realm whose path is dir or an ancestor of dir.
func (d *DB) Resolve(dir string) (Realm, bool, error) {
	realms, err := d.All()
	if err != nil {
		return Realm{}, false, err
	}
	var best Realm
	found := false
	for _, r := range realms {
		if dir == r.Path || strings.HasPrefix(dir, r.Path+string(filepath.Separator)) {
			if !found || len(r.Path) > len(best.Path) {
				best, found = r, true
			}
		}
	}
	return best, found, nil
}

// Delete removes a realm row.
func (d *DB) Delete(id int64) error {
	_, err := d.sql.Exec("DELETE FROM realms WHERE id = ?", id)
	if err == nil {
		d.dirty = true
	}
	return err
}

// SetPath re-points a realm at a new directory and clears any orphan flag.
func (d *DB) SetPath(id int64, path string) error {
	_, err := d.sql.Exec("UPDATE realms SET path = ?, orphaned_at = NULL WHERE id = ?", path, id)
	if err != nil {
		if isUniqueErr(err, "realms.path") {
			return ErrPathRegistered
		}
		return err
	}
	d.dirty = true
	return nil
}

// SetName renames a realm. The caller is responsible for renaming the
// derived history files (see HistFile).
func (d *DB) SetName(id int64, name string) error {
	_, err := d.sql.Exec("UPDATE realms SET name = ? WHERE id = ?", name, id)
	if err != nil {
		if isUniqueErr(err, "realms.name") {
			return ErrNameTaken
		}
		return err
	}
	d.dirty = true
	return nil
}

// TouchLastUsed updates last_used_at, but only when the stored value is
// older than staleAfter — avoids write churn on every directory change.
func (d *DB) TouchLastUsed(id int64, now time.Time, staleAfter time.Duration) error {
	_, err := d.sql.Exec(
		"UPDATE realms SET last_used_at = ? WHERE id = ? AND (last_used_at IS NULL OR last_used_at < ?)",
		now.Unix(), id, now.Add(-staleAfter).Unix(),
	)
	return err
}

// MarkOrphaned flags a realm whose directory is missing.
func (d *DB) MarkOrphaned(id int64, now time.Time) error {
	_, err := d.sql.Exec("UPDATE realms SET orphaned_at = ? WHERE id = ? AND orphaned_at IS NULL", now.Unix(), id)
	return err
}

// ClearOrphaned unflags a realm whose directory reappeared.
func (d *DB) ClearOrphaned(id int64) error {
	_, err := d.sql.Exec("UPDATE realms SET orphaned_at = NULL WHERE id = ?", id)
	return err
}

// HistFile returns the history file path for realm r and shell ("zsh"/"bash").
func (d *DB) HistFile(r Realm, shell string) string {
	name := fmt.Sprintf("%d-%s.%s.history", r.ID, paths.SanitizeName(r.Name), shell)
	return filepath.Join(d.dataDir, "histories", name)
}

func scanRealm(row interface{ Scan(...any) error }) (Realm, error) {
	var r Realm
	var created int64
	var lastUsed, orphaned sql.NullInt64
	if err := row.Scan(&r.ID, &r.Name, &r.Path, &created, &lastUsed, &orphaned); err != nil {
		return Realm{}, err
	}
	r.CreatedAt = time.Unix(created, 0)
	if lastUsed.Valid {
		r.LastUsedAt = time.Unix(lastUsed.Int64, 0)
	}
	if orphaned.Valid {
		r.OrphanedAt = time.Unix(orphaned.Int64, 0)
	}
	return r, nil
}

func isUniqueErr(err error, column string) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: "+column)
}
