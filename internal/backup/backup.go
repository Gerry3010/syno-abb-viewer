// Package backup models the server's dated backup runs and their contents.
//
// Runs are plain dated directories (see runs.go), each holding databases/ (gzipped
// SQL dumps), projects/, and volumes/. Dump inspection (table listing) lives in
// dump.go; run-vs-run comparison comes in a later milestone.
package backup
