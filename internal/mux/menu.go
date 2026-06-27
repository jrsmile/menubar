package mux

import (
	"strconv"

	"github.com/gdamore/tcell/v2"
)

// menuItem is a single clickable entry in the drop-down menu.
type menuItem struct {
	label  string
	action func()
}

// vt10x glyph attribute bits (the constants are unexported in the library, so
// we mirror their values here).
const (
	attrReverse   = 1 << 0
	attrUnderline = 1 << 1
	attrBold      = 1 << 2
	attrItalic    = 1 << 4
	attrBlink     = 1 << 5
)

const (
	menuButtonLabel = "[Menu]"
	xButtonLabel    = "[X]"
)

// buildMenu (re)computes the drop-down entries for the current pane set.
func (a *App) buildMenu() {
	items := []menuItem{
		{label: "New Pane", action: a.newPane},
	}
	for i, p := range a.panes {
		i := i
		label := "Go to Pane " + strconv.Itoa(p.ID())
		if i == a.active {
			label += " *"
		}
		items = append(items, menuItem{
			label:  label,
			action: func() { a.setActive(i) },
		})
	}
	items = append(items, menuItem{label: "Close Pane", action: a.closeActivePane})

	width := len(menuButtonLabel)
	for _, it := range items {
		if l := len(it.label) + 2; l > width {
			width = l
		}
	}

	a.menuItems = items
	a.menuWidth = width
}

// handleMouse routes a mouse press. Plain clicks on the top row drive the menu
// and the X button; plain clicks inside the pane are ignored so PuTTY's own
// Shift+click selection / paste keeps working (those events never reach us).
func (a *App) handleMouse(ev *tcell.EventMouse) {
	// Only act on a button press, not motion or release.
	if ev.Buttons()&(tcell.Button1|tcell.Button2|tcell.Button3) == 0 {
		return
	}

	x, y := ev.Position()
	w, _ := a.screen.Size()

	if y == 0 {
		a.handleMenuBarClick(x, w)
		return
	}

	if a.menuOpen {
		if !a.handleDropdownClick(x, y) {
			a.menuOpen = false
		}
		return
	}

	// Inside the pane: deliberately ignored.
}

func (a *App) handleMenuBarClick(x, w int) {
	// X button occupies the last len(xButtonLabel) columns.
	if x >= w-len(xButtonLabel) {
		a.closeActivePane()
		return
	}
	// Anywhere else on the bar toggles the menu.
	a.menuOpen = !a.menuOpen
	if a.menuOpen {
		a.buildMenu()
	}
}

// handleDropdownClick returns true if the click landed on a menu entry.
func (a *App) handleDropdownClick(x, y int) bool {
	idx := y - 1
	if x < 0 || x >= a.menuWidth || idx < 0 || idx >= len(a.menuItems) {
		return false
	}
	action := a.menuItems[idx].action
	a.menuOpen = false
	if action != nil {
		action()
	}
	return true
}
