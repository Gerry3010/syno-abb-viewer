package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/Gerry3010/syno-abb-viewer/internal/backup"
	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

// RunsView lists discovered backup runs on the left and, for the selected run,
// shows its date/size header plus a tree of its contents (databases/projects/
// volumes) on the right. Scanning and size computation run off the UI thread.
type RunsView struct {
	split  *container.Split
	list   *widget.List
	header *widget.Label
	detail *FileBrowser
	status func(string)

	fs   remotefs.FS
	root string
	runs []backup.Run
}

// NewRunsView builds an empty runs view. status feeds the shared status bar.
func NewRunsView(status func(string)) *RunsView {
	v := &RunsView{status: status}

	v.list = widget.NewList(
		func() int { return len(v.runs) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(v.runs[i].When.Format("2006-01-02  15:04:05"))
		},
	)
	v.list.OnSelected = v.onSelect

	v.header = widget.NewLabel("Not connected")
	v.detail = NewFileBrowser(status)

	right := container.NewBorder(v.header, nil, nil, nil, v.detail.Object())
	v.split = container.NewHSplit(v.list, right)
	v.split.SetOffset(0.32)
	return v
}

// Object returns the widget for placement in a container.
func (v *RunsView) Object() fyne.CanvasObject { return v.split }

// SetFS points the view at a connected filesystem and scans for runs.
func (v *RunsView) SetFS(fs remotefs.FS, root string) {
	v.fs = fs
	v.root = root
	v.runs = nil
	v.list.UnselectAll()
	v.list.Refresh()
	v.header.SetText("Scanning for backup runs …")

	go func() {
		runs, _ := backup.FindRuns(fs, root, 8)
		fyne.Do(func() {
			v.runs = runs
			v.list.Refresh()
			switch len(runs) {
			case 0:
				v.header.SetText("No backup runs found under " + root)
			default:
				v.header.SetText(fmt.Sprintf("%d backup run(s) — select one", len(runs)))
			}
		})
	}()
}

func (v *RunsView) onSelect(id widget.ListItemID) {
	if id < 0 || id >= len(v.runs) {
		return
	}
	r := v.runs[id]
	when := r.When.Format("2006-01-02 15:04:05")

	v.detail.SetFS(v.fs, r.Path)
	v.header.SetText(when + "   •   computing size …")

	fs := v.fs
	go func() {
		st, err := backup.RunStats(fs, r)
		fyne.Do(func() {
			if err != nil {
				v.header.SetText(when + "   •   size error: " + err.Error())
				return
			}
			v.header.SetText(fmt.Sprintf("%s   •   %s   •   %d files",
				when, humanizeBytes(st.Size), st.Files))
		})
	}()
}
