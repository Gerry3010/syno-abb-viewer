package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/Gerry3010/syno-abb-viewer/internal/backup"
	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

// showCompareDialog lets the user pick two runs and shows the file-level diff
// (added / removed / changed by size). Walking both run trees runs off the UI thread.
func showCompareDialog(win fyne.Window, fs remotefs.FS, runs []backup.Run) {
	if len(runs) < 2 {
		dialog.ShowInformation("Compare runs", "Need at least two backup runs to compare.", win)
		return
	}

	labels := make([]string, len(runs))
	for i, r := range runs {
		labels[i] = r.When.Format("2006-01-02 15:04:05")
	}

	selA := widget.NewSelect(labels, nil)
	selB := widget.NewSelect(labels, nil)
	selA.SetSelectedIndex(1) // older
	selB.SetSelectedIndex(0) // newer

	summary := widget.NewLabel("Pick two runs and press Compare.")
	summary.Wrapping = fyne.TextWrapWord

	var rows []string
	list := widget.NewList(
		func() int { return len(rows) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(rows[i]) },
	)

	compareBtn := widget.NewButton("Compare", nil)
	compareBtn.Importance = widget.HighImportance
	compareBtn.OnTapped = func() {
		ia, ib := selA.SelectedIndex(), selB.SelectedIndex()
		if ia < 0 || ib < 0 {
			return
		}
		if ia == ib {
			dialog.ShowInformation("Compare runs", "Pick two different runs.", win)
			return
		}
		a, b := runs[ia], runs[ib]
		summary.SetText("Comparing …")
		rows = nil
		list.Refresh()
		compareBtn.Disable()
		go func() {
			diff, err := backup.CompareRuns(fs, a, b)
			fyne.Do(func() {
				compareBtn.Enable()
				if err != nil {
					summary.SetText("Error: " + err.Error())
					return
				}
				rows = formatChanges(diff.Changes)
				list.Refresh()
				summary.SetText(fmt.Sprintf("%s → %s   •   +%d added, -%d removed, ~%d changed, %d unchanged",
					a.When.Format("01-02 15:04"), b.When.Format("01-02 15:04"),
					diff.AddedN, diff.RemovedN, diff.ChangedN, diff.Unchanged))
			})
		}()
	}

	picker := container.NewGridWithColumns(2,
		container.NewVBox(widget.NewLabel("Run A (base)"), selA),
		container.NewVBox(widget.NewLabel("Run B (target)"), selB),
	)
	top := container.NewVBox(picker, container.NewHBox(compareBtn), widget.NewSeparator(), summary)

	content := container.NewBorder(top, nil, nil, nil, list)
	d := dialog.NewCustom("Compare runs", "Close", content, win)
	d.Resize(fyne.NewSize(680, 600))
	d.Show()
}

// formatChanges renders each change as a monospace "+/-/~ relpath  sizes" line.
func formatChanges(changes []backup.FileChange) []string {
	rows := make([]string, 0, len(changes))
	for _, c := range changes {
		switch c.Kind {
		case backup.Added:
			rows = append(rows, fmt.Sprintf("+ %s   (%s)", c.RelPath, humanizeBytes(c.NewSize)))
		case backup.Removed:
			rows = append(rows, fmt.Sprintf("- %s   (%s)", c.RelPath, humanizeBytes(c.OldSize)))
		case backup.Changed:
			rows = append(rows, fmt.Sprintf("~ %s   (%s → %s)", c.RelPath,
				humanizeBytes(c.OldSize), humanizeBytes(c.NewSize)))
		}
	}
	return rows
}
