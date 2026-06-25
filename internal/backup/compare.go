package backup

import (
	"sort"
	"strings"

	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

// ChangeKind classifies how a file differs between two runs.
type ChangeKind int

const (
	Added   ChangeKind = iota // present in B, absent in A
	Removed                   // present in A, absent in B
	Changed                   // present in both, different size
)

func (k ChangeKind) String() string {
	switch k {
	case Added:
		return "added"
	case Removed:
		return "removed"
	default:
		return "changed"
	}
}

// FileChange is one differing file, identified by its path relative to the run.
type FileChange struct {
	RelPath string
	Kind    ChangeKind
	OldSize int64 // for Removed/Changed
	NewSize int64 // for Added/Changed
}

// Diff is the comparison of two runs (A = older/base, B = newer/target by
// convention, though the caller picks). Changes lists only differing files.
type Diff struct {
	A, B      Run
	Changes   []FileChange
	Unchanged int
	AddedN    int
	RemovedN  int
	ChangedN  int
}

// CompareRuns diffs two runs by relative path and file size. Content is not
// hashed (that would mean downloading every file over SFTP); size is a good
// proxy, and for SQL dumps a size change reliably flags a changed dump.
func CompareRuns(fsys remotefs.FS, a, b Run) (Diff, error) {
	amap, err := fileMap(fsys, a.Path)
	if err != nil {
		return Diff{}, err
	}
	bmap, err := fileMap(fsys, b.Path)
	if err != nil {
		return Diff{}, err
	}

	diff := Diff{A: a, B: b}
	for rel, ae := range amap {
		be, ok := bmap[rel]
		if !ok {
			diff.Changes = append(diff.Changes, FileChange{RelPath: rel, Kind: Removed, OldSize: ae.Size})
			diff.RemovedN++
			continue
		}
		if ae.Size != be.Size {
			diff.Changes = append(diff.Changes, FileChange{RelPath: rel, Kind: Changed, OldSize: ae.Size, NewSize: be.Size})
			diff.ChangedN++
		} else {
			diff.Unchanged++
		}
	}
	for rel, be := range bmap {
		if _, ok := amap[rel]; !ok {
			diff.Changes = append(diff.Changes, FileChange{RelPath: rel, Kind: Added, NewSize: be.Size})
			diff.AddedN++
		}
	}

	sort.Slice(diff.Changes, func(i, j int) bool {
		if diff.Changes[i].Kind != diff.Changes[j].Kind {
			return diff.Changes[i].Kind < diff.Changes[j].Kind
		}
		return diff.Changes[i].RelPath < diff.Changes[j].RelPath
	})
	return diff, nil
}

// fileMap walks a run and keys every file by its path relative to the run root.
func fileMap(fsys remotefs.FS, root string) (map[string]remotefs.Entry, error) {
	files, err := remotefs.WalkFiles(fsys, root)
	if err != nil {
		return nil, err
	}
	m := make(map[string]remotefs.Entry, len(files))
	for _, e := range files {
		rel := strings.TrimPrefix(e.Path, root)
		rel = strings.TrimPrefix(rel, "/")
		m[rel] = e
	}
	return m, nil
}
