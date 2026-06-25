package backup

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

// DefaultDumpScan caps how many *decompressed* bytes InspectDump reads when no
// explicit limit is given. pg_dump writes all schema (CREATE TABLE …) before the
// data, so the table list is captured well within this without streaming a
// multi-GB dump in full over SFTP.
const DefaultDumpScan = 32 << 20 // 32 MiB

// createTableRe matches "CREATE TABLE [IF NOT EXISTS] <name>" and captures the
// (optionally schema-qualified, optionally quoted) table name.
var createTableRe = regexp.MustCompile(`(?i)^\s*CREATE TABLE\s+(?:IF NOT EXISTS\s+)?([^\s(]+)`)

// DumpInfo is the result of inspecting a gzipped SQL dump.
type DumpInfo struct {
	Tables       []string // table names in first-seen order
	ScannedBytes int64    // decompressed bytes read
	Truncated    bool     // scan stopped at the limit before EOF
}

// InspectDump streams a gzipped SQL dump and lists the tables it declares.
// It stops after maxScan decompressed bytes (use 0 for unlimited / full scan).
func InspectDump(fsys remotefs.FS, path string, maxScan int64) (DumpInfo, error) {
	rc, err := fsys.Open(path)
	if err != nil {
		return DumpInfo{}, err
	}
	defer rc.Close()

	gz, err := gzip.NewReader(rc)
	if err != nil {
		return DumpInfo{}, fmt.Errorf("gunzip %s: %w", path, err)
	}
	defer gz.Close()

	counter := &countingReader{r: gz}
	sc := bufio.NewScanner(counter)
	sc.Buffer(make([]byte, 64*1024), 8<<20) // tolerate long lines (wide COPY rows)

	var info DumpInfo
	seen := map[string]bool{}
	for sc.Scan() {
		if m := createTableRe.FindStringSubmatch(sc.Text()); m != nil {
			// Normalize quoted identifiers: "public"."orders" -> public.orders.
			name := strings.ReplaceAll(m[1], `"`, "")
			if !seen[name] {
				seen[name] = true
				info.Tables = append(info.Tables, name)
			}
		}
		if maxScan > 0 && counter.n >= maxScan {
			info.Truncated = true
			break
		}
	}
	info.ScannedBytes = counter.n

	if err := sc.Err(); err != nil {
		// A huge data row past the schema can exceed the line buffer — by then we
		// already have the tables, so treat it as a truncated (still useful) scan.
		if errors.Is(err, bufio.ErrTooLong) {
			info.Truncated = true
			return info, nil
		}
		return info, err
	}
	return info, nil
}

// countingReader tallies the bytes read so the scan can be capped by volume.
type countingReader struct {
	r interface{ Read([]byte) (int, error) }
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
