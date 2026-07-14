package cleanup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Positronico/yahh/internal/registry"
)

var now = time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

const grace = 30 * 24 * time.Hour

func setup(t *testing.T) (*registry.DB, string) {
	t.Helper()
	dataDir := t.TempDir()
	db, err := registry.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db, dataDir
}

func TestScanLifecycle(t *testing.T) {
	db, _ := setup(t)
	live := t.TempDir()
	gone := filepath.Join(t.TempDir(), "gone")

	if _, err := db.Create("live", live, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Create("gone", gone, now); err != nil {
		t.Fatal(err)
	}

	// First scan: the missing directory is marked, nothing expires.
	rep, err := Scan(db, now, grace, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Marked) != 1 || rep.Marked[0].Name != "gone" {
		t.Fatalf("Marked = %+v", rep.Marked)
	}
	if len(rep.Expired) != 0 || len(rep.Healed) != 0 {
		t.Fatalf("unexpected report %+v", rep)
	}

	// Within grace: still nothing to remove.
	rep, _ = Scan(db, now.Add(grace/2), grace, true)
	if len(rep.Expired) != 0 {
		t.Fatalf("expired too early: %+v", rep.Expired)
	}

	// Past grace: eligible for removal.
	rep, _ = Scan(db, now.Add(grace+time.Hour), grace, true)
	if len(rep.Expired) != 1 || rep.Expired[0].Name != "gone" {
		t.Fatalf("Expired = %+v", rep.Expired)
	}

	// Directory comes back: healed, flag cleared.
	if err := os.MkdirAll(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	rep, _ = Scan(db, now.Add(grace+2*time.Hour), grace, true)
	if len(rep.Healed) != 1 || rep.Healed[0].Name != "gone" {
		t.Fatalf("Healed = %+v", rep.Healed)
	}
	r, _, _ := db.ByName("gone")
	if r.Orphaned() {
		t.Error("orphan flag not cleared after heal")
	}
}

func TestScanDryRunDoesNotWrite(t *testing.T) {
	db, _ := setup(t)
	gone := filepath.Join(t.TempDir(), "gone")
	if _, err := db.Create("gone", gone, now); err != nil {
		t.Fatal(err)
	}
	if _, err := Scan(db, now, grace, false); err != nil {
		t.Fatal(err)
	}
	r, _, _ := db.ByName("gone")
	if r.Orphaned() {
		t.Error("dry-run scan wrote the orphan flag")
	}
}

func TestRemoveArchivesHistories(t *testing.T) {
	db, dataDir := setup(t)
	r, err := db.Create("p", filepath.Join(t.TempDir(), "p"), now)
	if err != nil {
		t.Fatal(err)
	}
	hist := db.HistFile(r, "zsh")
	os.MkdirAll(filepath.Dir(hist), 0o700)
	os.WriteFile(hist, []byte("echo hi\n"), 0o600)

	archived, err := Remove(db, r, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(archived) != 1 {
		t.Fatalf("archived = %v", archived)
	}
	if _, err := os.Stat(hist); !os.IsNotExist(err) {
		t.Error("live history file still present")
	}
	data, err := os.ReadFile(archived[0])
	if err != nil || string(data) != "echo hi\n" {
		t.Errorf("archived content = %q, %v", data, err)
	}
	if filepath.Dir(archived[0]) != filepath.Join(dataDir, "archive") {
		t.Errorf("archive location = %s", archived[0])
	}
	if _, ok, _ := db.ByName("p"); ok {
		t.Error("realm row still present")
	}
}

func TestRemovePurge(t *testing.T) {
	db, dataDir := setup(t)
	r, _ := db.Create("p", filepath.Join(t.TempDir(), "p"), now)
	hist := db.HistFile(r, "zsh")
	os.MkdirAll(filepath.Dir(hist), 0o700)
	os.WriteFile(hist, []byte("echo hi\n"), 0o600)

	archived, err := Remove(db, r, true, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(archived) != 0 {
		t.Errorf("purge archived files: %v", archived)
	}
	entries, _ := os.ReadDir(filepath.Join(dataDir, "archive"))
	if len(entries) != 0 {
		t.Errorf("archive dir not empty after purge")
	}
}

func TestPurgeArchive(t *testing.T) {
	_, dataDir := setup(t)
	archDir := filepath.Join(dataDir, "archive")
	os.MkdirAll(archDir, 0o700)
	old := filepath.Join(archDir, "old.history.removed.x")
	os.WriteFile(old, []byte("x"), 0o600)
	oldTime := now.Add(-100 * 24 * time.Hour)
	os.Chtimes(old, oldTime, oldTime)
	recent := filepath.Join(archDir, "recent.history.removed.x")
	os.WriteFile(recent, []byte("x"), 0o600)
	os.Chtimes(recent, now, now)

	purged, err := PurgeArchive(dataDir, 90*24*time.Hour, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(purged) != 1 || purged[0] != old {
		t.Errorf("purged = %v", purged)
	}
	if _, err := os.Stat(recent); err != nil {
		t.Error("recent archive file was deleted")
	}
}
