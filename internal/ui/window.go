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
	browser := NewFileBrowser(func(s string) { status.SetText(s) })

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
		browser.SetFS(remotefs.NewSFTP(conn.SFTP), c.RootPath)
		status.SetText("Connected to " + c.User + "@" + c.Host)
	}

	connectBtn := widget.NewButtonWithIcon("Connect", theme.LoginIcon(), func() {
		showConnectDialog(w, cfg, onConnect)
	})
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), browser.Reload)

	top := container.NewHBox(connectBtn, refreshBtn)
	content := container.NewBorder(top, status, nil, nil, browser.Object())
	w.SetContent(content)

	w.SetCloseIntercept(func() {
		closeCurrent()
		w.Close()
	})
	return w
}
