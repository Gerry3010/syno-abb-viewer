package ui

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

type nopWriteCloser struct{ *bytes.Buffer }

func (nopWriteCloser) Close() error { return nil }

func gz(s string) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write([]byte(s))
	w.Close()
	return buf.Bytes()
}

func TestStreamToWriterRaw(t *testing.T) {
	fs := remotefs.NewMapFS().AddFile("/dir/file.txt", []byte("hello world"))
	var out bytes.Buffer
	n, err := streamToWriter(fs, "/dir/file.txt", nopWriteCloser{&out}, false)
	if err != nil {
		t.Fatalf("streamToWriter: %v", err)
	}
	if n != 11 || out.String() != "hello world" {
		t.Fatalf("raw copy wrong: n=%d out=%q", n, out.String())
	}
}

func TestStreamToWriterExtract(t *testing.T) {
	fs := remotefs.NewMapFS().AddFile("/dir/db.sql.gz", gz("CREATE TABLE t (id int);"))
	var out bytes.Buffer
	n, err := streamToWriter(fs, "/dir/db.sql.gz", nopWriteCloser{&out}, true)
	if err != nil {
		t.Fatalf("streamToWriter extract: %v", err)
	}
	if got := out.String(); got != "CREATE TABLE t (id int);" {
		t.Fatalf("extract wrong: %q", got)
	}
	if n != int64(out.Len()) {
		t.Fatalf("byte count mismatch: n=%d len=%d", n, out.Len())
	}
}

func TestStreamToWriterExtractBadGzip(t *testing.T) {
	fs := remotefs.NewMapFS().AddFile("/dir/notgz.gz", []byte("not gzip data"))
	var out bytes.Buffer
	if _, err := streamToWriter(fs, "/dir/notgz.gz", nopWriteCloser{&out}, true); err == nil {
		t.Fatal("expected gunzip error on non-gzip data")
	}
}
