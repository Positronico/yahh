package registry

import (
	"database/sql"
	"errors"
	"strconv"
	"time"
)

func (d *DB) getMeta(key string) (string, bool, error) {
	var v string
	err := d.sql.QueryRow("SELECT value FROM meta WHERE key = ?", key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

func (d *DB) setMeta(key, value string) error {
	_, err := d.sql.Exec(
		"INSERT INTO meta(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	return err
}

// Enabled reports whether realm switching is globally enabled (default true).
func (d *DB) Enabled() (bool, error) {
	v, ok, err := d.getMeta("enabled")
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}
	return v != "0", nil
}

// SetEnabled persists the global enabled flag.
func (d *DB) SetEnabled(enabled bool) error {
	v := "1"
	if !enabled {
		v = "0"
	}
	return d.setMeta("enabled", v)
}

// LastCleanAt returns when clean last ran (zero time if never).
func (d *DB) LastCleanAt() (time.Time, error) {
	v, ok, err := d.getMeta("last_clean_at")
	if err != nil || !ok {
		return time.Time{}, err
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return time.Time{}, nil
	}
	return time.Unix(n, 0), nil
}

// SetLastCleanAt records a clean run.
func (d *DB) SetLastCleanAt(t time.Time) error {
	return d.setMeta("last_clean_at", strconv.FormatInt(t.Unix(), 10))
}

// ClaimAutoClean atomically claims the right to run an auto-clean: it
// succeeds for exactly one caller per interval, even across concurrent
// shell startups, by updating last_clean_at only when it is stale.
func (d *DB) ClaimAutoClean(now time.Time, interval time.Duration) (bool, error) {
	res, err := d.sql.Exec(`
INSERT INTO meta(key, value) VALUES('last_clean_at', ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value
WHERE CAST(meta.value AS INTEGER) <= ?`,
		strconv.FormatInt(now.Unix(), 10), now.Add(-interval).Unix(),
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
