package ui

import (
	"fmt"
	"path"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

// FileBrowser is a lazily-loaded directory tree backed by a remotefs.FS.
//
// Children are fetched on branch-open in a goroutine (SFTP is latency-prone) and
// cached; the widget's synchronous callbacks only ever read the cache. UI updates
// from the fetch goroutine are marshalled back with fyne.Do.
type FileBrowser struct {
	tree   *widget.Tree
	status func(string)

	// OnSelectFile, if set, is called when a non-directory node is selected,
	// with the browser's filesystem and the selected entry.
	OnSelectFile func(fs remotefs.FS, e remotefs.Entry)

	mu       sync.Mutex
	fs       remotefs.FS
	root     string
	children map[string][]remotefs.Entry // dir path -> direct children
	meta     map[string]remotefs.Entry   // path -> metadata
	loaded   map[string]bool             // dirs whose children are cached or loading
}

// NewFileBrowser builds an empty browser. status receives one-line progress and
// error messages for the status bar.
func NewFileBrowser(status func(string)) *FileBrowser {
	b := &FileBrowser{
		status:   status,
		children: map[string][]remotefs.Entry{},
		meta:     map[string]remotefs.Entry{},
		loaded:   map[string]bool{},
	}
	b.tree = widget.NewTree(b.childUIDs, b.isBranch, createNode, b.updateNode)
	// Loading is driven by childUIDs (auto-load on first ask), not OnBranchOpened,
	// so a Refresh re-populates every visible branch, not just the root.
	b.tree.OnSelected = func(id widget.TreeNodeID) {
		b.mu.Lock()
		e, ok := b.meta[id]
		fs := b.fs
		b.mu.Unlock()
		if ok && !e.IsDir && b.OnSelectFile != nil {
			b.OnSelectFile(fs, e)
		}
	}
	return b
}

// Object returns the underlying widget for placement in a container.
func (b *FileBrowser) Object() fyne.CanvasObject { return b.tree }

// SetFS swaps in a new filesystem rooted at root, clears caches, and loads the root.
func (b *FileBrowser) SetFS(fs remotefs.FS, root string) {
	b.mu.Lock()
	b.fs = fs
	b.root = root
	b.children = map[string][]remotefs.Entry{}
	b.meta = map[string]remotefs.Entry{}
	b.loaded = map[string]bool{}
	b.mu.Unlock()

	b.tree.Root = root
	b.tree.Refresh()
	b.load(root)
}

// Reload re-reads the currently connected filesystem from the root.
func (b *FileBrowser) Reload() {
	b.mu.Lock()
	fs, root := b.fs, b.root
	b.mu.Unlock()
	if fs != nil {
		b.SetFS(fs, root)
	}
}

func (b *FileBrowser) childUIDs(id widget.TreeNodeID) []widget.TreeNodeID {
	b.mu.Lock()
	key := id
	if key == "" {
		key = b.root
	}
	kids := b.children[key]
	needLoad := b.fs != nil && !b.loaded[key]
	ids := make([]widget.TreeNodeID, 0, len(kids))
	for _, e := range kids {
		ids = append(ids, e.Path)
	}
	b.mu.Unlock()

	// Auto-load on first ask. load() is idempotent and does no synchronous UI
	// work, so triggering it from within this render callback is safe.
	if needLoad {
		b.load(key)
	}
	return ids
}

func (b *FileBrowser) isBranch(id widget.TreeNodeID) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if id == "" || id == b.root {
		return true
	}
	if e, ok := b.meta[id]; ok {
		return e.IsDir
	}
	return false
}

// createNode builds a row template: name on the left, right-aligned size.
func createNode(bool) fyne.CanvasObject {
	name := widget.NewLabel("")
	size := widget.NewLabel("")
	size.Alignment = fyne.TextAlignTrailing
	return container.NewHBox(name, layout.NewSpacer(), size)
}

func (b *FileBrowser) updateNode(id widget.TreeNodeID, _ bool, obj fyne.CanvasObject) {
	row := obj.(*fyne.Container)
	name := row.Objects[0].(*widget.Label)
	size := row.Objects[2].(*widget.Label)

	name.SetText(path.Base(id))

	b.mu.Lock()
	e, ok := b.meta[id]
	b.mu.Unlock()
	if ok && !e.IsDir {
		size.SetText(humanizeBytes(e.Size))
	} else {
		size.SetText("")
	}
}

// load fetches a directory's children once, off the UI thread.
func (b *FileBrowser) load(dir string) {
	b.mu.Lock()
	if b.fs == nil || b.loaded[dir] {
		b.mu.Unlock()
		return
	}
	b.loaded[dir] = true
	fs := b.fs
	b.mu.Unlock()

	// No synchronous UI here: load() may be called from within a render pass
	// (childUIDs). All UI mutations happen on the UI thread via fyne.Do.
	go func() {
		entries, err := fs.ReadDir(dir)
		if err != nil {
			b.mu.Lock()
			b.loaded[dir] = false // allow retry
			b.mu.Unlock()
			fyne.Do(func() { b.setStatus("Error: " + err.Error()) })
			return
		}
		b.mu.Lock()
		b.children[dir] = entries
		for _, e := range entries {
			b.meta[e.Path] = e
		}
		b.mu.Unlock()
		fyne.Do(func() {
			b.tree.Refresh()
			b.setStatus(fmt.Sprintf("%d items in %s", len(entries), dir))
		})
	}()
}

func (b *FileBrowser) setStatus(s string) {
	if b.status != nil {
		b.status(s)
	}
}

// humanizeBytes formats a byte count as a short human-readable string.
func humanizeBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
