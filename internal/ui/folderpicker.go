package ui

import (
	"path"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

// showRemoteFolderPicker opens a directory-only tree over fs, lets the user pick
// a folder, and calls onPick with its path. onClosed runs when the dialog is
// dismissed (select or cancel) — the caller uses it to close the connection.
func showRemoteFolderPicker(win fyne.Window, fs remotefs.FS, start string, onPick func(string), onClosed func()) {
	if start == "" {
		start = "/"
	}

	children := map[string][]widget.TreeNodeID{}
	loaded := map[string]bool{}
	selected := start

	var tree *widget.Tree
	load := func(dir string) {
		if loaded[dir] {
			return
		}
		loaded[dir] = true
		go func() {
			entries, err := fs.ReadDir(dir)
			if err != nil {
				return
			}
			kids := make([]widget.TreeNodeID, 0, len(entries))
			for _, e := range entries {
				if e.IsDir {
					kids = append(kids, e.Path)
				}
			}
			fyne.Do(func() {
				children[dir] = kids
				tree.Refresh()
			})
		}()
	}

	tree = widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			if id == "" {
				id = start
			}
			return children[id]
		},
		func(widget.TreeNodeID) bool { return true }, // every node is a (potential) dir branch
		func(bool) fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TreeNodeID, _ bool, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(path.Base(id))
		},
	)
	tree.Root = start
	tree.OnBranchOpened = load
	load(start)

	chosen := widget.NewLabel(start)
	tree.OnSelected = func(id widget.TreeNodeID) {
		selected = id
		chosen.SetText(id)
	}

	var d *dialog.CustomDialog
	selectBtn := widget.NewButton("Select folder", func() {
		onPick(selected)
		d.Hide()
	})
	selectBtn.Importance = widget.HighImportance
	cancelBtn := widget.NewButton("Cancel", func() { d.Hide() })

	bottom := container.NewVBox(
		container.NewHBox(widget.NewLabel("Selected:"), chosen),
		container.NewHBox(layout.NewSpacer(), cancelBtn, selectBtn),
	)
	content := container.NewBorder(nil, bottom, nil, nil, tree)

	d = dialog.NewCustomWithoutButtons("Pick a folder on the DiskStation", content, win)
	d.SetOnClosed(onClosed)
	d.Resize(fyne.NewSize(480, 480))
	d.Show()
}
