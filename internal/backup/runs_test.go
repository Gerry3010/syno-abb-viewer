package backup

import (
	"testing"

	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

// buildShare mirrors the real ABB layout: a task dir whose
// var/backups/synology/<timestamp>/ folders are the runs.
func buildShare() *remotefs.MapFS {
	base := "/volume1/abb/ionos-server-neu/var/backups/synology"
	return remotefs.NewMapFS().
		AddFile(base+"/2026-06-25_00-47-45/databases/passbubble.sql.gz", make([]byte, 100)).
		AddFile(base+"/2026-06-25_00-47-45/databases/pipepush.sql.gz", make([]byte, 50)).
		AddFile(base+"/2026-06-25_00-47-45/projects/opt_passbubble/compose.yml", make([]byte, 25)).
		AddDir(base + "/2026-06-25_02-00-01").
		AddFile(base+"/backup.log", make([]byte, 10))
}

func TestFindRunsNewestFirst(t *testing.T) {
	runs, err := FindRuns(buildShare(), "/volume1", 8)
	if err != nil {
		t.Fatalf("FindRuns: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("want 2 runs, got %d: %+v", len(runs), runs)
	}
	if runs[0].Name != "2026-06-25_02-00-01" {
		t.Fatalf("runs not newest-first: %s before %s", runs[0].Name, runs[1].Name)
	}
	if runs[0].When.Before(runs[1].When) {
		t.Fatal("When ordering wrong")
	}
}

func TestFindRunsIgnoresNonRunDirs(t *testing.T) {
	// backup.log is a file and "synology" is not a timestamp — neither is a run.
	runs, _ := FindRuns(buildShare(), "/volume1", 8)
	for _, r := range runs {
		if _, ok := parseRunTime(r.Name); !ok {
			t.Fatalf("non-run leaked in: %q", r.Name)
		}
	}
}

func TestRunStats(t *testing.T) {
	runs, _ := FindRuns(buildShare(), "/volume1", 8)
	var older Run
	for _, r := range runs {
		if r.Name == "2026-06-25_00-47-45" {
			older = r
		}
	}
	st, err := RunStats(buildShare(), older)
	if err != nil {
		t.Fatalf("RunStats: %v", err)
	}
	if st.Size != 175 || st.Files != 3 {
		t.Fatalf("want size=175 files=3, got size=%d files=%d", st.Size, st.Files)
	}
}

func TestParseRunTimeRejectsJunk(t *testing.T) {
	for _, bad := range []string{"databases", "2026-06-25", "backup.log", "2026-13-99_00-00-00"} {
		if _, ok := parseRunTime(bad); ok {
			t.Fatalf("parseRunTime accepted junk: %q", bad)
		}
	}
}
