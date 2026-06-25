package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Gerry3010/syno-abb-viewer/internal/config"
	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
	"github.com/Gerry3010/syno-abb-viewer/internal/sshconn"
)

// NewMainWindow builds the main application window: a Connect/Refresh bar, the
// lazily-loaded backup tree, and a status line along the bottom.
func NewMainWindow(a fyne.App) fyne.Window {
	w := a.NewWindow("syno-abb-viewer")
	w.Resize(fyne.NewSize(900, 640))

	cfg, _ := config.Load()

	status := widget.NewLabel("Not connected")
	setStatus := func(s string) { status.SetText(s) }

	runs := NewRunsView(w, setStatus)
	files := NewFileBrowser(setStatus)
	files.OnSelectFile = func(fs remotefs.FS, e remotefs.Entry) {
		if isDumpFile(e.Name) {
			showDumpInspector(w, fs, e)
		}
	}

	var current *sshconn.Conn
	closeCurrent := func() {
		if current != nil {
			current.Close()
			current = nil
		}
	}

	onConnect := func(conn *sshconn.Conn, c config.Config) {
		closeCurrent()
		current = conn
		cfg = c
		fs := remotefs.NewSFTP(conn.SFTP)
		runs.SetFS(fs, c.RootPath)
		files.SetFS(fs, c.RootPath)
		status.SetText("Connected to " + c.User + "@" + c.Host)
	}

	connectBtn := widget.NewButtonWithIcon("Connect", theme.LoginIcon(), func() {
		showConnectDialog(w, cfg, onConnect)
	})
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		if current == nil {
			return
		}
		runs.SetFS(runs.fs, runs.root)
		files.Reload()
	})
	compareBtn := widget.NewButtonWithIcon("Compare", theme.ContentCopyIcon(), func() {
		if current == nil {
			return
		}
		showCompareDialog(w, runs.fs, runs.runs)
	})

	tabs := container.NewAppTabs(
		container.NewTabItem("Runs", runs.Object()),
		container.NewTabItem("Files", files.Object()),
	)

	top := container.NewHBox(connectBtn, refreshBtn, compareBtn)
	content := container.NewBorder(top, status, nil, nil, tabs)
	w.SetContent(content)

	w.SetCloseIntercept(func() {
		closeCurrent()
		w.Close()
	})
	return w
}
