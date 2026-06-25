// Package remotefs abstracts the remote backup filesystem the UI browses.
//
// The FS interface is the one seam in the app that earns an interface: it lets
// the Fyne tree be unit-tested against an in-memory fake (MapFS) without a live
// DiskStation, and keeps the UI free of any sftp import.
package remotefs

import (
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pkg/sftp"
)

// Entry describes one file or directory.
type Entry struct {
	Name    string
	Path    string // absolute, forward-slash path on the remote host
	IsDir   bool
	Size    int64
	ModTime time.Time
}

// FS is a read-only view of a remote filesystem.
type FS interface {
	// ReadDir lists the direct children of dir, directories first then by name.
	ReadDir(dir string) ([]Entry, error)
	// Stat returns metadata for a single path.
	Stat(p string) (Entry, error)
	// Open opens a file for reading (used by later milestones: download/inspect).
	Open(p string) (io.ReadCloser, error)
}

// sortEntries orders directories before files, then case-insensitively by name.
func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
}

// sftpFS implements FS over an SFTP client.
type sftpFS struct {
	c *sftp.Client
}

// NewSFTP wraps an sftp client as an FS.
func NewSFTP(c *sftp.Client) FS { return &sftpFS{c: c} }

func (f *sftpFS) ReadDir(dir string) ([]Entry, error) {
	infos, err := f.c.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(infos))
	for _, fi := range infos {
		entries = append(entries, Entry{
			Name:    fi.Name(),
			Path:    path.Join(dir, fi.Name()),
			IsDir:   fi.IsDir(),
			Size:    fi.Size(),
			ModTime: fi.ModTime(),
		})
	}
	sortEntries(entries)
	return entries, nil
}

func (f *sftpFS) Stat(p string) (Entry, error) {
	fi, err := f.c.Stat(p)
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		Name:    path.Base(p),
		Path:    p,
		IsDir:   fi.IsDir(),
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
	}, nil
}

func (f *sftpFS) Open(p string) (io.ReadCloser, error) {
	return f.c.Open(p)
}
