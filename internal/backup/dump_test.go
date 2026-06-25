package backup

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

const sampleDump = `--
-- PostgreSQL database dump
--
SET statement_timeout = 0;

CREATE TABLE public.users (
    id integer NOT NULL,
    email text
);

CREATE TABLE IF NOT EXISTS "public"."orders" (
    id integer NOT NULL
);

COPY public.users (id, email) FROM stdin;
1	a@b.c
\.

CREATE TABLE analytics.events (id bigint);
`

func gzipped(s string) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write([]byte(s))
	w.Close()
	return buf.Bytes()
}

func TestInspectDumpListsTables(t *testing.T) {
	fsys := remotefs.NewMapFS().AddFile("/run/databases/db.sql.gz", gzipped(sampleDump))

	info, err := InspectDump(fsys, "/run/databases/db.sql.gz", DefaultDumpScan)
	if err != nil {
		t.Fatalf("InspectDump: %v", err)
	}
	want := []string{"public.users", `public.orders`, "analytics.events"}
	if len(info.Tables) != len(want) {
		t.Fatalf("want %d tables, got %v", len(want), info.Tables)
	}
	for i := range want {
		if info.Tables[i] != want[i] {
			t.Fatalf("table %d: want %q, got %q", i, want[i], info.Tables[i])
		}
	}
	if info.Truncated {
		t.Fatal("small dump should not be truncated")
	}
}

func TestInspectDumpTruncates(t *testing.T) {
	fsys := remotefs.NewMapFS().AddFile("/run/databases/db.sql.gz", gzipped(sampleDump))
	// A tiny cap forces truncation before the whole dump is read.
	info, err := InspectDump(fsys, "/run/databases/db.sql.gz", 16)
	if err != nil {
		t.Fatalf("InspectDump: %v", err)
	}
	if !info.Truncated {
		t.Fatal("expected truncated scan with a 16-byte cap")
	}
}

func TestInspectDumpMissing(t *testing.T) {
	fsys := remotefs.NewMapFS()
	if _, err := InspectDump(fsys, "/nope.sql.gz", DefaultDumpScan); err == nil {
		t.Fatal("expected error opening missing dump")
	}
}
