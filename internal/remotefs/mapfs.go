package remotefs

import (
	"bytes"
	"io"
	"io/fs"
	"path"
	"time"
)

// MapFS is an in-memory FS used for tests. It is not used by the running app.
type MapFS struct {
	dirs  map[string]bool   // set of directory paths that exist
	files map[string][]byte // file path -> contents
	mtime time.Time
}

// NewMapFS returns an empty in-memory FS. The root "/" always exists.
func NewMapFS() *MapFS {
	return &MapFS{
		dirs:  map[string]bool{"/": true},
		files: map[string][]byte{},
		mtime: time.Unix(0, 0),
	}
}

// AddDir registers a directory (and its ancestors).
func (m *MapFS) AddDir(p string) *MapFS {
	for d := path.Clean(p); d != "/" && d != "."; d = path.Dir(d) {
		m.dirs[d] = true
	}
	return m
}

// AddFile registers a file (and its ancestor directories) with the given content.
func (m *MapFS) AddFile(p string, content []byte) *MapFS {
	m.AddDir(path.Dir(p))
	m.files[path.Clean(p)] = content
	return m
}

func (m *MapFS) ReadDir(dir string) ([]Entry, error) {
	dir = path.Clean(dir)
	if !m.dirs[dir] {
		return nil, &fs.PathError{Op: "readdir", Path: dir, Err: fs.ErrNotExist}
	}
	seen := map[string]bool{}
	var entries []Entry
	add := func(full string, isDir bool, size int64) {
		if path.Dir(full) != dir || seen[full] {
			return
		}
		seen[full] = true
		entries = append(entries, Entry{
			Name:    path.Base(full),
			Path:    full,
			IsDir:   isDir,
			Size:    size,
			ModTime: m.mtime,
		})
	}
	for d := range m.dirs {
		add(d, true, 0)
	}
	for f, c := range m.files {
		add(f, false, int64(len(c)))
	}
	sortEntries(entries)
	return entries, nil
}

func (m *MapFS) Stat(p string) (Entry, error) {
	p = path.Clean(p)
	if m.dirs[p] {
		return Entry{Name: path.Base(p), Path: p, IsDir: true, ModTime: m.mtime}, nil
	}
	if c, ok := m.files[p]; ok {
		return Entry{Name: path.Base(p), Path: p, Size: int64(len(c)), ModTime: m.mtime}, nil
	}
	return Entry{}, &fs.PathError{Op: "stat", Path: p, Err: fs.ErrNotExist}
}

func (m *MapFS) Open(p string) (io.ReadCloser, error) {
	c, ok := m.files[path.Clean(p)]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: p, Err: fs.ErrNotExist}
	}
	return io.NopCloser(bytes.NewReader(c)), nil
}
