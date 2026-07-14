// Package registry is the SQLite-backed store mapping directories to realms.
package registry

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB is the yahh realm registry.
type DB struct {
	sql     *sql.DB
	dataDir string
}

// Open opens (creating if needed) the registry database under dataDir.
func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.Join(dataDir, "yahh.db") +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// A single connection serializes writers within this process and keeps
	// the pool from opening a connection per query on the resolve hot path.
	sqlDB.SetMaxOpenConns(1)
	if err := migrate(sqlDB); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return &DB{sql: sqlDB, dataDir: dataDir}, nil
}

// Close closes the underlying database.
func (d *DB) Close() error { return d.sql.Close() }

// DataDir returns the data directory this registry lives in.
func (d *DB) DataDir() string { return d.dataDir }

// Vacuum checkpoints the WAL and compacts the database file.
func (d *DB) Vacuum() error {
	if _, err := d.sql.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return err
	}
	_, err := d.sql.Exec("VACUUM")
	return err
}

const schemaVersion = 1

func migrate(db *sql.DB) error {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= schemaVersion {
		return nil
	}
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS realms (
  id           INTEGER PRIMARY KEY,
  name         TEXT    NOT NULL UNIQUE,
  path         TEXT    NOT NULL UNIQUE,
  created_at   INTEGER NOT NULL,
  last_used_at INTEGER,
  orphaned_at  INTEGER
);
CREATE INDEX IF NOT EXISTS realms_path_idx ON realms(path);
CREATE TABLE IF NOT EXISTS meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
PRAGMA user_version = 1;
`)
	return err
}
