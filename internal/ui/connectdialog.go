package ui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/Gerry3010/syno-abb-viewer/internal/config"
	"github.com/Gerry3010/syno-abb-viewer/internal/sshconn"
)

// showConnectDialog presents the connection form, pre-filled from cfg. On a
// successful connect it calls onConnect with the live connection and the config
// that produced it. Dialing and host-key prompts run off the UI thread.
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

	keyItem := widget.NewFormItem("Key path", keyPath)
	// Toggle the key-path row depending on auth method.
	syncAuth := func(sel string) {
		if sel == "Password" {
			keyPath.Disable()
		} else {
			keyPath.Enable()
		}
	}
	auth.OnChanged = syncAuth
	syncAuth(auth.Selected)

	items := []*widget.FormItem{
		widget.NewFormItem("Host", host),
		widget.NewFormItem("Port", port),
		widget.NewFormItem("User", user),
		widget.NewFormItem("Auth", auth),
		keyItem,
		widget.NewFormItem("Password", password),
		widget.NewFormItem("Root path", root),
	}

	form := dialog.NewForm("Connect to DiskStation", "Connect", "Cancel", items, func(ok bool) {
		if !ok {
			return
		}
		p, err := strconv.Atoi(port.Text)
		if err != nil || p < 1 || p > 65535 {
			dialog.ShowError(fmt.Errorf("invalid port: %q", port.Text), win)
			return
		}
		newCfg := cfg
		newCfg.Host = host.Text
		newCfg.Port = p
		newCfg.User = user.Text
		newCfg.KeyPath = keyPath.Text
		newCfg.RootPath = root.Text
		newCfg.Password = password.Text
		if auth.Selected == "Password" {
			newCfg.Auth = config.AuthPassword
		} else {
			newCfg.Auth = config.AuthKey
		}

		// Persist the non-secret fields (Password has json:"-").
		if err := config.Save(newCfg); err != nil {
			dialog.ShowError(fmt.Errorf("save config: %w", err), win)
		}

		dialConnection(win, newCfg, onConnect)
	}, win)

	form.Resize(fyne.NewSize(480, 360))
	form.Show()
}

// dialConnection dials in the background, prompting for unknown host keys, and
// reports the outcome on the UI thread.
func dialConnection(win fyne.Window, cfg config.Config, onConnect func(*sshconn.Conn, config.Config)) {
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
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			onConnect(conn, cfg)
		})
	}()
}
