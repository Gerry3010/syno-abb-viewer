// Package backup models the server's dated backup runs and their contents.
//
// Runs are plain dated directories (see runs.go). Each holds databases/ (dumps),
// projects/, and volumes/. Per-dump table listing and run-vs-run comparison come
// in later milestones; Dump is the placeholder for that work.
package backup

// Dump is a database dump file within a run's databases/ directory
// (e.g. "passbubble-postgres-1_2026-06-25_00-47-45.sql.gz"). Table listing is
// added in a later milestone.
type Dump struct {
	Path string
	Size int64
}
