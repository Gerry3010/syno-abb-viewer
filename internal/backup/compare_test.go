package backup

import (
	"testing"

	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

func TestCompareRuns(t *testing.T) {
	a := "/runs/2026-06-25_00-47-45"
	b := "/runs/2026-06-25_02-00-01"
	fsys := remotefs.NewMapFS().
		// unchanged
		AddFile(a+"/databases/pipepush.sql.gz", make([]byte, 100)).
		AddFile(b+"/databases/pipepush.sql.gz", make([]byte, 100)).
		// changed (size differs)
		AddFile(a+"/databases/passbubble.sql.gz", make([]byte, 500)).
		AddFile(b+"/databases/passbubble.sql.gz", make([]byte, 800)).
		// removed (only in A)
		AddFile(a+"/projects/old_app/compose.yml", make([]byte, 20)).
		// added (only in B)
		AddFile(b+"/projects/new_app/compose.yml", make([]byte, 30))

	runA := Run{Name: "A", Path: a}
	runB := Run{Name: "B", Path: b}

	diff, err := CompareRuns(fsys, runA, runB)
	if err != nil {
		t.Fatalf("CompareRuns: %v", err)
	}
	if diff.AddedN != 1 || diff.RemovedN != 1 || diff.ChangedN != 1 || diff.Unchanged != 1 {
		t.Fatalf("counts wrong: added=%d removed=%d changed=%d unchanged=%d",
			diff.AddedN, diff.RemovedN, diff.ChangedN, diff.Unchanged)
	}
	if len(diff.Changes) != 3 {
		t.Fatalf("want 3 changes, got %d", len(diff.Changes))
	}

	// Find the changed dump and verify the size delta is reported.
	var found bool
	for _, c := range diff.Changes {
		if c.RelPath == "databases/passbubble.sql.gz" {
			found = true
			if c.Kind != Changed || c.OldSize != 500 || c.NewSize != 800 {
				t.Fatalf("changed entry wrong: %+v", c)
			}
		}
	}
	if !found {
		t.Fatal("changed dump not in diff")
	}
}

func TestCompareRunsIdentical(t *testing.T) {
	a := "/runs/x"
	b := "/runs/y"
	fsys := remotefs.NewMapFS().
		AddFile(a+"/db.sql.gz", make([]byte, 42)).
		AddFile(b+"/db.sql.gz", make([]byte, 42))

	diff, _ := CompareRuns(fsys, Run{Path: a}, Run{Path: b})
	if len(diff.Changes) != 0 || diff.Unchanged != 1 {
		t.Fatalf("identical runs should have no changes: %+v", diff)
	}
}
