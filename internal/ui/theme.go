package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// terminalTheme is a dark, monospace "terminal" theme. It forces the monospace
// font for every text style and overrides the core colors for a dark backdrop
// with a green accent, delegating sizes, icons, and untouched colors to Fyne's
// dark base theme.
type terminalTheme struct{}

// NewTerminalTheme returns the app's dark terminal theme.
func NewTerminalTheme() fyne.Theme { return terminalTheme{} }

var (
	colBackground = color.NRGBA{R: 0x0d, G: 0x11, B: 0x17, A: 0xff} // near-black
	colForeground = color.NRGBA{R: 0xc9, G: 0xd1, B: 0xd9, A: 0xff} // light grey
	colPrimary    = color.NRGBA{R: 0x3f, G: 0xb9, B: 0x50, A: 0xff} // terminal green
	colInputBg    = color.NRGBA{R: 0x16, G: 0x1b, B: 0x22, A: 0xff}
)

func (terminalTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colBackground
	case theme.ColorNameForeground:
		return colForeground
	case theme.ColorNamePrimary, theme.ColorNameHyperlink:
		return colPrimary
	case theme.ColorNameInputBackground, theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return colInputBg
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0x1f, G: 0x6f, B: 0x3a, A: 0xff}
	}
	// Everything else: Fyne's dark base, regardless of the OS light/dark setting.
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

// Font forces monospace for all styles so the whole UI reads like a terminal.
func (terminalTheme) Font(style fyne.TextStyle) fyne.Resource {
	style.Monospace = true
	return theme.DefaultTheme().Font(style)
}

func (terminalTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (terminalTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
