package backup

import (
	"regexp"
	"sort"
	"time"

	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

// runTimeLayout is the directory-name format the server's backup script uses,
// e.g. "2026-06-25_00-47-45".
const runTimeLayout = "2006-01-02_15-04-05"

var runNameRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2}$`)

// Run is a single dated backup run discovered on disk.
type Run struct {
	Name string    // the directory name (the raw timestamp)
	Path string    // absolute path of the run directory
	When time.Time // parsed from Name
}

// parseRunTime reports whether name is a run-timestamp directory and its time.
func parseRunTime(name string) (time.Time, bool) {
	if !runNameRe.MatchString(name) {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(runTimeLayout, name, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// FindRuns walks fsys from root (down to maxDepth) collecting directories whose
// names are run timestamps, newest first. It does not descend into a run once
// found, and skips subtrees it can't read.
func FindRuns(fsys remotefs.FS, root string, maxDepth int) ([]Run, error) {
	var runs []Run
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if depth > maxDepth {
			return
		}
		entries, err := fsys.ReadDir(dir)
		if err != nil {
			return // unreadable branch — skip
		}
		for _, e := range entries {
			if !e.IsDir {
				continue
			}
			if t, ok := parseRunTime(e.Name); ok {
				runs = append(runs, Run{Name: e.Name, Path: e.Path, When: t})
				continue // a run's contents aren't themselves runs
			}
			walk(e.Path, depth+1)
		}
	}
	walk(root, 0)
	sort.Slice(runs, func(i, j int) bool { return runs[i].When.After(runs[j].When) })
	return runs, nil
}

// Stats is the computed size summary of a run.
type Stats struct {
	Size  int64
	Files int
}

// RunStats sums the recursive size and file count of a run's directory.
func RunStats(fsys remotefs.FS, r Run) (Stats, error) {
	size, files, err := remotefs.DirSize(fsys, r.Path)
	return Stats{Size: size, Files: files}, err
}
