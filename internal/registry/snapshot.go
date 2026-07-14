package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The snapshot is a tiny shell-readable mirror of the registry that lets
// the shell hooks resolve realms in pure shell (no process spawn on the
// cd hot path). Format:
//
//	yahh1 enabled=<0|1>
//	<realm root>\t<zsh histfile>\t<bash histfile>
//	...
//
// Realms are ordered deepest root first so the shell's first prefix match
// is the correct (most specific) one. The binary rewrites the file after
// every mutation; the hooks fall back to `yahh resolve` when it is absent.

// SnapshotPath returns the location of the shell-readable snapshot.
func (d *DB) SnapshotPath() string { return filepath.Join(d.dataDir, "snapshot") }

// WriteSnapshot atomically (re)writes the snapshot file.
func (d *DB) WriteSnapshot() error {
	enabled, err := d.Enabled()
	if err != nil {
		return err
	}
	realms, err := d.All()
	if err != nil {
		return err
	}
	sort.SliceStable(realms, func(i, j int) bool { return len(realms[i].Path) > len(realms[j].Path) })

	var b strings.Builder
	flag := "1"
	if !enabled {
		flag = "0"
	}
	fmt.Fprintf(&b, "yahh1 enabled=%s\n", flag)
	for _, r := range realms {
		fmt.Fprintf(&b, "%s\t%s\t%s\n", r.Path, d.HistFile(r, "zsh"), d.HistFile(r, "bash"))
	}

	tmp, err := os.CreateTemp(d.dataDir, "snapshot-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(b.String()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), d.SnapshotPath())
}

// EnsureSnapshot writes the snapshot if the file does not exist yet
// (self-healing after upgrades or a deleted file).
func (d *DB) EnsureSnapshot() error {
	if _, err := os.Stat(d.SnapshotPath()); err == nil {
		return nil
	}
	return d.WriteSnapshot()
}
