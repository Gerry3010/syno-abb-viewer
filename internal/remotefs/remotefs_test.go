package remotefs

import (
	"io"
	"testing"
)

// buildTree mirrors a small slice of a backup layout.
func buildTree() *MapFS {
	return NewMapFS().
		AddDir("/volume1/backups/2026-06-24").
		AddFile("/volume1/backups/2026-06-24/db.sql", []byte("CREATE TABLE t;")).
		AddFile("/volume1/backups/2026-06-24/files.tar", make([]byte, 2048)).
		AddDir("/volume1/backups/2026-06-25")
}

func TestReadDirSortsDirsFirstThenName(t *testing.T) {
	fsys := buildTree()
	entries, err := fsys.ReadDir("/volume1/backups/2026-06-24")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	// Both are files here; check name order and sizes.
	if entries[0].Name != "db.sql" || entries[1].Name != "files.tar" {
		t.Fatalf("unexpected order: %s, %s", entries[0].Name, entries[1].Name)
	}
	if entries[1].Size != 2048 {
		t.Fatalf("want size 2048, got %d", entries[1].Size)
	}
}

func TestReadDirDirectoriesBeforeFiles(t *testing.T) {
	fsys := buildTree()
	entries, err := fsys.ReadDir("/volume1/backups")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 || !entries[0].IsDir || !entries[1].IsDir {
		t.Fatalf("expected two run directories, got %+v", entries)
	}
	if entries[0].Name != "2026-06-24" || entries[1].Name != "2026-06-25" {
		t.Fatalf("runs out of order: %+v", entries)
	}
}

func TestStatAndOpen(t *testing.T) {
	fsys := buildTree()
	st, err := fsys.Stat("/volume1/backups/2026-06-24/db.sql")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if st.IsDir || st.Name != "db.sql" {
		t.Fatalf("unexpected stat: %+v", st)
	}
	rc, err := fsys.Open("/volume1/backups/2026-06-24/db.sql")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if string(data) != "CREATE TABLE t;" {
		t.Fatalf("unexpected content: %q", data)
	}
}

func TestReadDirMissing(t *testing.T) {
	if _, err := buildTree().ReadDir("/nope"); err == nil {
		t.Fatal("expected error for missing dir")
	}
}

// walk exercises the same navigation pattern the tree widget uses: descend
// directory-by-directory via ReadDir, never walking the whole tree up front.
func TestWalkNavigation(t *testing.T) {
	fsys := buildTree()
	var files int
	var walk func(dir string)
	walk = func(dir string) {
		entries, err := fsys.ReadDir(dir)
		if err != nil {
			t.Fatalf("ReadDir(%s): %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir {
				walk(e.Path)
			} else {
				files++
			}
		}
	}
	walk("/volume1")
	if files != 2 {
		t.Fatalf("expected 2 files in tree, found %d", files)
	}
}
