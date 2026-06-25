package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Gerry3010/syno-abb-viewer/internal/config"
	"github.com/Gerry3010/syno-abb-viewer/internal/remotefs"
	"github.com/Gerry3010/syno-abb-viewer/internal/sshconn"
)

// showConnectDialog presents the connection form, pre-filled from cfg, with
// Try (test only), Save (persist only), and Connect actions. On a successful
// Connect it calls onConnect with the live connection and the config that
// produced it. Dialing and host-key prompts run off the UI thread.
func showConnectDialog(win fyne.Window, cfg config.Config, onConnect func(*sshconn.Conn, config.Config)) {
	host := widget.NewEntry()
	host.SetText(cfg.Host)
	host.SetPlaceHolder("diskstation.local or 192.168.1.10")

	port := widget.NewEntry()
	port.SetText(strconv.Itoa(cfg.Port))

	user := widget.NewEntry()
	user.SetText(cfg.User)

	keyPath := widget.NewEntry()
	keyPath.SetText(cfg.KeyPath)

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("password, or passphrase for an encrypted key")

	root := widget.NewEntry()
	root.SetText(cfg.RootPath)

	auth := widget.NewRadioGroup([]string{"SSH key", "Password"}, nil)
	if cfg.Auth == config.AuthPassword {
		auth.SetSelected("Password")
	} else {
		auth.SetSelected("SSH key")
	}

	// Browse button: pick the SSH key file, starting in ~/.ssh.
	browse := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil || rc == nil {
				return
			}
			defer rc.Close()
			keyPath.SetText(rc.URI().Path())
		}, win)
		if home, err := os.UserHomeDir(); err == nil {
			if lister, err := storage.ListerForURI(storage.NewFileURI(filepath.Join(home, ".ssh"))); err == nil {
				fd.SetLocation(lister)
			}
		}
		fd.Show()
	})
	keyRow := container.NewBorder(nil, nil, nil, browse, keyPath)

	// Toggle the key row depending on auth method.
	syncAuth := func(sel string) {
		if sel == "Password" {
			keyPath.Disable()
			browse.Disable()
		} else {
			keyPath.Enable()
			browse.Enable()
		}
	}
	auth.OnChanged = syncAuth
	syncAuth(auth.Selected)

	// readCfg gathers the current field values into a Config, validating the port.
	readCfg := func() (config.Config, error) {
		p, err := strconv.Atoi(port.Text)
		if err != nil || p < 1 || p > 65535 {
			return config.Config{}, fmt.Errorf("invalid port: %q", port.Text)
		}
		c := cfg
		c.Host = host.Text
		c.Port = p
		c.User = user.Text
		c.KeyPath = keyPath.Text
		c.RootPath = root.Text
		c.Password = password.Text
		if auth.Selected == "Password" {
			c.Auth = config.AuthPassword
		} else {
			c.Auth = config.AuthKey
		}
		return c, nil
	}

	// Root browse: connect with the current settings, then pick a folder on the
	// NAS over SFTP (Fyne's own file dialog can only browse the local machine).
	rootBrowse := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		c, err := readCfg()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		dialAsync(win, c, func(conn *sshconn.Conn, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			showRemoteFolderPicker(win, remotefs.NewSFTP(conn.SFTP), root.Text,
				func(p string) { root.SetText(p) },
				func() { conn.Close() },
			)
		})
	})
	rootRow := container.NewBorder(nil, nil, nil, rootBrowse, root)

	form := widget.NewForm(
		widget.NewFormItem("Host", host),
		widget.NewFormItem("Port", port),
		widget.NewFormItem("User", user),
		widget.NewFormItem("Auth", auth),
		widget.NewFormItem("Key path", keyRow),
		widget.NewFormItem("Password", password),
		widget.NewFormItem("Root path", rootRow),
	)

	var d *dialog.CustomDialog

	tryBtn := widget.NewButtonWithIcon("Try", theme.SearchIcon(), func() {
		c, err := readCfg()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		dialAsync(win, c, func(conn *sshconn.Conn, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			conn.Close() // test only — drop the connection
			dialog.ShowInformation("Connection OK", "Successfully connected to "+c.User+"@"+c.Host, win)
		})
	})

	saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		c, err := readCfg()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		if err := config.Save(c); err != nil { // Password has json:"-"
			dialog.ShowError(fmt.Errorf("save config: %w", err), win)
			return
		}
		dialog.ShowInformation("Saved", "Connection settings saved.", win)
	})

	connectBtn := widget.NewButtonWithIcon("Connect", theme.ConfirmIcon(), func() {
		c, err := readCfg()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		if err := config.Save(c); err != nil {
			dialog.ShowError(fmt.Errorf("save config: %w", err), win)
		}
		dialAsync(win, c, func(conn *sshconn.Conn, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			d.Hide()
			onConnect(conn, c)
		})
	})
	connectBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButton("Cancel", func() { d.Hide() })

	buttons := container.NewHBox(layout.NewSpacer(), tryBtn, saveBtn, cancelBtn, connectBtn)
	content := container.NewVBox(form, widget.NewSeparator(), buttons)

	d = dialog.NewCustomWithoutButtons("Connect to DiskStation", content, win)
	d.Resize(fyne.NewSize(540, 440))
	d.Show()
}

// dialAsync dials cfg in the background, prompting for unknown host keys, and
// delivers the result (connection or error) on the UI thread.
func dialAsync(win fyne.Window, cfg config.Config, onResult func(*sshconn.Conn, error)) {
	progress := dialog.NewCustomWithoutButtons("Connecting", widget.NewProgressBarInfinite(), win)
	progress.Show()

	trust := func(host, fingerprint string) bool {
		ch := make(chan bool, 1)
		fyne.Do(func() {
			msg := fmt.Sprintf("Unknown host %s\n\nSHA256 fingerprint:\n%s\n\nTrust this host and remember it?", host, fingerprint)
			dialog.NewConfirm("Verify host key", msg, func(ok bool) { ch <- ok }, win).Show()
		})
		return <-ch
	}

	go func() {
		conn, err := sshconn.Dial(cfg, trust)
		fyne.Do(func() {
			progress.Hide()
			onResult(conn, err)
		})
	}()
}
