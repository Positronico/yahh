package registry

import (
	"sync"
	"testing"
	"time"
)

func openTest(t *testing.T) *DB {
	t.Helper()
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

var now = time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

func TestCreateAndNameSuffixing(t *testing.T) {
	db := openTest(t)
	r1, err := db.Create("proj", "/a/proj", now)
	if err != nil {
		t.Fatal(err)
	}
	if r1.Name != "proj" {
		t.Errorf("name = %q", r1.Name)
	}
	r2, err := db.Create("proj", "/b/proj", now)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Name != "proj-2" {
		t.Errorf("second name = %q, want proj-2", r2.Name)
	}
	if _, err := db.Create("other", "/a/proj", now); err != ErrPathRegistered {
		t.Errorf("duplicate path error = %v, want ErrPathRegistered", err)
	}
}

func TestResolveLongestPrefix(t *testing.T) {
	db := openTest(t)
	mustCreate := func(name, path string) Realm {
		r, err := db.Create(name, path, now)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	outer := mustCreate("outer", "/a/foo")
	inner := mustCreate("inner", "/a/foo/sub")
	mustCreate("bar", "/a/foobar")

	cases := []struct {
		dir  string
		want string
		ok   bool
	}{
		{"/a/foo", outer.Name, true},
		{"/a/foo/x/y", outer.Name, true},
		{"/a/foo/sub", inner.Name, true},
		{"/a/foo/sub/deep", inner.Name, true},
		{"/a/foobar", "bar", true},
		{"/a/foobarbaz", "", false}, // boundary: /a/foobar must not match its sibling
		{"/a", "", false},
		{"/elsewhere", "", false},
	}
	for _, c := range cases {
		r, ok, err := db.Resolve(c.dir)
		if err != nil {
			t.Fatal(err)
		}
		if ok != c.ok || (ok && r.Name != c.want) {
			t.Errorf("Resolve(%q) = (%q, %v), want (%q, %v)", c.dir, r.Name, ok, c.want, c.ok)
		}
	}
}

func TestTouchLastUsedThrottled(t *testing.T) {
	db := openTest(t)
	r, err := db.Create("p", "/p", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.TouchLastUsed(r.ID, now, time.Hour); err != nil {
		t.Fatal(err)
	}
	// A touch within the stale window must not move the timestamp.
	if err := db.TouchLastUsed(r.ID, now.Add(30*time.Minute), time.Hour); err != nil {
		t.Fatal(err)
	}
	got, _, err := db.ByName("p")
	if err != nil {
		t.Fatal(err)
	}
	if !got.LastUsedAt.Equal(time.Unix(now.Unix(), 0)) {
		t.Errorf("last_used_at moved within stale window: %v", got.LastUsedAt)
	}
	// After the window it must move.
	later := now.Add(2 * time.Hour)
	if err := db.TouchLastUsed(r.ID, later, time.Hour); err != nil {
		t.Fatal(err)
	}
	got, _, _ = db.ByName("p")
	if !got.LastUsedAt.Equal(time.Unix(later.Unix(), 0)) {
		t.Errorf("last_used_at = %v, want %v", got.LastUsedAt, later)
	}
}

func TestOrphanFlagging(t *testing.T) {
	db := openTest(t)
	r, _ := db.Create("p", "/p", now)
	if err := db.MarkOrphaned(r.ID, now); err != nil {
		t.Fatal(err)
	}
	// Marking again must not refresh the original timestamp.
	if err := db.MarkOrphaned(r.ID, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, _, _ := db.ByName("p")
	if !got.OrphanedAt.Equal(time.Unix(now.Unix(), 0)) {
		t.Errorf("orphaned_at = %v, want %v", got.OrphanedAt, now)
	}
	if err := db.ClearOrphaned(r.ID); err != nil {
		t.Fatal(err)
	}
	got, _, _ = db.ByName("p")
	if got.Orphaned() {
		t.Error("orphan flag not cleared")
	}
}

func TestEnabledDefaultsTrue(t *testing.T) {
	db := openTest(t)
	enabled, err := db.Enabled()
	if err != nil || !enabled {
		t.Fatalf("Enabled() = (%v, %v), want (true, nil)", enabled, err)
	}
	if err := db.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if enabled, _ = db.Enabled(); enabled {
		t.Error("still enabled after SetEnabled(false)")
	}
}

func TestClaimAutoClean(t *testing.T) {
	db := openTest(t)
	interval := 7 * 24 * time.Hour

	claimed, err := db.ClaimAutoClean(now, interval)
	if err != nil || !claimed {
		t.Fatalf("first claim = (%v, %v), want (true, nil)", claimed, err)
	}
	claimed, err = db.ClaimAutoClean(now.Add(time.Hour), interval)
	if err != nil || claimed {
		t.Fatalf("claim within interval = (%v, %v), want (false, nil)", claimed, err)
	}
	claimed, err = db.ClaimAutoClean(now.Add(interval+time.Hour), interval)
	if err != nil || !claimed {
		t.Fatalf("claim after interval = (%v, %v), want (true, nil)", claimed, err)
	}
}

func TestClaimAutoCleanRace(t *testing.T) {
	db := openTest(t)
	const n = 8
	var wg sync.WaitGroup
	wins := make(chan bool, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := db.ClaimAutoClean(now, 7*24*time.Hour)
			if err != nil {
				t.Error(err)
				return
			}
			wins <- claimed
		}()
	}
	wg.Wait()
	close(wins)
	won := 0
	for w := range wins {
		if w {
			won++
		}
	}
	if won != 1 {
		t.Errorf("%d concurrent claims won, want exactly 1", won)
	}
}

func TestSetNameAndSetPath(t *testing.T) {
	db := openTest(t)
	r1, _ := db.Create("a", "/a", now)
	db.Create("b", "/b", now)
	if err := db.SetName(r1.ID, "b"); err != ErrNameTaken {
		t.Errorf("SetName to taken name = %v, want ErrNameTaken", err)
	}
	if err := db.SetPath(r1.ID, "/b"); err != ErrPathRegistered {
		t.Errorf("SetPath to taken path = %v, want ErrPathRegistered", err)
	}
	if err := db.SetPath(r1.ID, "/c"); err != nil {
		t.Fatal(err)
	}
	got, _, _ := db.ByName("a")
	if got.Path != "/c" {
		t.Errorf("path = %q, want /c", got.Path)
	}
}
