// Package backup will model Synology backup runs and database dumps.
//
// Milestone 1 only browses the raw directory tree, so these types are
// placeholders to be fleshed out in later milestones (view runs, inspect
// dumps, compare runs). They are intentionally minimal for now.
package backup

import "time"

// Status is the outcome of a backup run.
type Status string

const (
	StatusUnknown Status = "unknown"
	StatusOK      Status = "ok"
	StatusFailed  Status = "failed"
)

// Run is a single backup run discovered on the DiskStation. Fields will be
// derived from the on-disk layout once that convention is pinned down.
type Run struct {
	Path   string
	When   time.Time
	Size   int64
	Status Status
}

// Dump is a database dump file within a run (tables/size come later).
type Dump struct {
	Path string
	Size int64
}
