package ui

import (
	"compress/gzip"
	"fmt"
	"io"
	"path"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
)

// openFileAction is the FileBrowser selection handler: dumps open the inspector
// (which can also download/extract), other files get a small download sheet.
func openFileAction(win fyne.Window, fs remotefs.FS, e remotefs.Entry) {
	if isDumpFile(e.Name) {
		showDumpInspector(win, fs, e)
		return
	}
	showFileActions(win, fs, e)
}

// showFileActions offers Download (and Extract for .gz) for a single file.
func showFileActions(win fyne.Window, fs remotefs.FS, entry remotefs.Entry) {
	name := path.Base(entry.Path)
	info := widget.NewLabel(fmt.Sprintf("%s\n%s", name, humanizeBytes(entry.Size)))

	var d *dialog.CustomDialog
	buttons := []fyne.CanvasObject{
		widget.NewButtonWithIcon("Download", theme.DownloadIcon(), func() {
			d.Hide()
			downloadFile(win, fs, entry, false)
		}),
	}
	if strings.HasSuffix(strings.ToLower(name), ".gz") {
		buttons = append(buttons, widget.NewButtonWithIcon("Extract (gunzip)", theme.DownloadIcon(), func() {
			d.Hide()
			downloadFile(win, fs, entry, true)
		}))
	}

	content := container.NewVBox(info, widget.NewSeparator(), container.NewHBox(buttons...))
	d = dialog.NewCustom("File", "Close", content, win)
	d.Show()
}

// downloadFile streams a remote file to a locally chosen path, optionally
// gunzipping it on the way (extract). The copy runs off the UI thread.
func downloadFile(win fyne.Window, fs remotefs.FS, entry remotefs.Entry, extract bool) {
	defaultName := path.Base(entry.Path)
	if extract {
		defaultName = strings.TrimSuffix(defaultName, ".gz")
	}

	save := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		if wc == nil {
			return // cancelled
		}
		progress := dialog.NewCustomWithoutButtons("Downloading "+defaultName+" …", widget.NewProgressBarInfinite(), win)
		progress.Show()
		dest := wc.URI().Path()
		go func() {
			n, err := streamToWriter(fs, entry.Path, wc, extract)
			fyne.Do(func() {
				progress.Hide()
				if err != nil {
					dialog.ShowError(err, win)
					return
				}
				dialog.ShowInformation("Downloaded",
					fmt.Sprintf("%s\n%s written to\n%s", path.Base(entry.Path), humanizeBytes(n), dest), win)
			})
		}()
	}, win)
	save.SetFileName(defaultName)
	save.Show()
}

// streamToWriter copies the remote file into wc, optionally decompressing gzip.
// It always closes wc.
func streamToWriter(fs remotefs.FS, remotePath string, wc io.WriteCloser, extract bool) (int64, error) {
	defer wc.Close()

	rc, err := fs.Open(remotePath)
	if err != nil {
		return 0, err
	}
	defer rc.Close()

	var src io.Reader = rc
	if extract {
		gz, err := gzip.NewReader(rc)
		if err != nil {
			return 0, fmt.Errorf("gunzip: %w", err)
		}
		defer gz.Close()
		src = gz
	}
	return io.Copy(wc, src)
}
