// Command syno-abb-viewer is a desktop GUI for browsing Synology DiskStation
// backups over SSH/SFTP.
package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/Gerry3010/syno-abb-viewer/internal/ui"
)

func main() {
	a := app.NewWithID("com.gerry3010.syno-abb-viewer")
	a.Settings().SetTheme(ui.NewTerminalTheme())

	w := ui.NewMainWindow(a)
	w.ShowAndRun()
}
