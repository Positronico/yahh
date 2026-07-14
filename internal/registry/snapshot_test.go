package registry

import (
	"os"
	"strings"
	"testing"
)

func readSnapshot(t *testing.T, db *DB) string {
	t.Helper()
	data, err := os.ReadFile(db.SnapshotPath())
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSnapshotWrittenOnCloseAfterMutation(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Create("proj", "/a/proj", now); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	snap := readSnapshot(t, db)
	lines := strings.Split(strings.TrimRight(snap, "\n"), "\n")
	if lines[0] != "yahh1 enabled=1" {
		t.Errorf("header = %q", lines[0])
	}
	if len(lines) != 2 {
		t.Fatalf("snapshot lines = %d, want 2:\n%s", len(lines), snap)
	}
	fields := strings.Split(lines[1], "\t")
	if len(fields) != 3 || fields[0] != "/a/proj" ||
		!strings.HasSuffix(fields[1], ".zsh.history") || !strings.HasSuffix(fields[2], ".bash.history") {
		t.Errorf("realm line = %q", lines[1])
	}
}

func TestSnapshotDeepestRootFirst(t *testing.T) {
	db := openTest(t)
	db.Create("outer", "/a", now)
	db.Create("inner", "/a/deep/nested", now)
	db.Create("mid", "/a/deep", now)
	if err := db.WriteSnapshot(); err != nil {
		t.Fatal(err)
	}
	snap := readSnapshot(t, db)
	lines := strings.Split(strings.TrimRight(snap, "\n"), "\n")[1:]
	roots := make([]string, len(lines))
	for i, l := range lines {
		roots[i] = strings.SplitN(l, "\t", 2)[0]
	}
	want := []string{"/a/deep/nested", "/a/deep", "/a"}
	for i := range want {
		if roots[i] != want[i] {
			t.Fatalf("roots = %v, want %v", roots, want)
		}
	}
}

func TestSnapshotDisabledFlag(t *testing.T) {
	db := openTest(t)
	if err := db.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if err := db.WriteSnapshot(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(readSnapshot(t, db), "yahh1 enabled=0\n") {
		t.Error("disabled flag not reflected in snapshot header")
	}
}

func TestEnsureSnapshot(t *testing.T) {
	db := openTest(t)
	if err := db.EnsureSnapshot(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(db.SnapshotPath()); err != nil {
		t.Fatal("snapshot not created")
	}
	// Existing snapshot must not be rewritten.
	if err := os.WriteFile(db.SnapshotPath(), []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := db.EnsureSnapshot(); err != nil {
		t.Fatal(err)
	}
	if readSnapshot(t, db) != "sentinel" {
		t.Error("EnsureSnapshot overwrote an existing snapshot")
	}
}

func TestReadOnlyCloseDoesNotWriteSnapshot(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	db.All()
	db.Enabled()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(db.SnapshotPath()); !os.IsNotExist(err) {
		t.Error("read-only use created a snapshot on Close")
	}
}
