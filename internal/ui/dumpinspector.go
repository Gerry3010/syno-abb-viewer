package ui

import (
	"fmt"
	"path"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Gerry3010/syno-abb-viewer/internal/backup"
	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

// isDumpFile reports whether name looks like a gzipped SQL dump.
func isDumpFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".sql.gz")
}

// showDumpInspector opens a dialog that streams a gzipped SQL dump and lists the
// tables it declares. It scans a capped prefix first (fast) and offers a full scan.
func showDumpInspector(win fyne.Window, fs remotefs.FS, entry remotefs.Entry) {
	name := path.Base(entry.Path)

	header := widget.NewLabel("Scanning " + name + " …")
	header.Wrapping = fyne.TextWrapWord

	var tables []string
	list := widget.NewList(
		func() int { return len(tables) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(tables[i]) },
	)

	fullBtn := widget.NewButtonWithIcon("Full scan", theme.SearchReplaceIcon(), nil)
	closeBtn := widget.NewButton("Close", nil)

	scan := func(maxScan int64) {
		header.SetText("Scanning " + name + " …")
		fullBtn.Disable()
		go func() {
			info, err := backup.InspectDump(fs, entry.Path, maxScan)
			fyne.Do(func() {
				fullBtn.Enable()
				if err != nil {
					header.SetText("Error: " + err.Error())
					return
				}
				tables = info.Tables
				list.Refresh()
				summary := fmt.Sprintf("%s   •   %s   •   %d tables",
					name, humanizeBytes(entry.Size), len(info.Tables))
				if info.Truncated {
					summary += fmt.Sprintf("\nPartial scan (first %s decompressed) — use Full scan for the rest.",
						humanizeBytes(info.ScannedBytes))
				}
				header.SetText(summary)
			})
		}()
	}

	fullBtn.OnTapped = func() { scan(0) }

	buttons := container.NewHBox(layout.NewSpacer(), fullBtn, closeBtn)
	content := container.NewBorder(header, buttons, nil, nil, list)

	d := dialog.NewCustomWithoutButtons("DB dump: "+name, content, win)
	closeBtn.OnTapped = d.Hide
	d.Resize(fyne.NewSize(620, 540))
	d.Show()

	scan(backup.DefaultDumpScan)
}
