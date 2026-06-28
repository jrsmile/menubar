package mux

import (
	"strconv"

	"github.com/gdamore/tcell/v2"

	"menubar/internal/config"
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
	cmdButtonLabel  = "[Commands]"
	xButtonLabel    = "[X]"
	cmdBackLabel    = ".."
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

// hasCmdMenu reports whether a user-defined command menu is available.
func (a *App) hasCmdMenu() bool { return len(a.cmdRoot) > 0 }

// cmdCurrentEntries resolves the submenu level currently addressed by cmdPath.
func (a *App) cmdCurrentEntries() []config.MenuEntry {
	entries := a.cmdRoot
	for _, idx := range a.cmdPath {
		if idx < 0 || idx >= len(entries) {
			return nil
		}
		entries = entries[idx].Submenu
	}
	return entries
}

// buildCmdMenu (re)computes the width of the command drop-down for the level
// currently addressed by cmdPath.
func (a *App) buildCmdMenu() {
	width := len(cmdButtonLabel)
	if len(a.cmdPath) > 0 {
		if l := len(cmdBackLabel) + 2; l > width {
			width = l
		}
	}
	for _, e := range a.cmdCurrentEntries() {
		label := e.Label
		if len(e.Submenu) > 0 {
			label += " >"
		}
		if l := len(label) + 2; l > width {
			width = l
		}
	}
	a.cmdMenuWidth = width
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

	if a.cmdMenuOpen {
		if !a.handleCmdDropdownClick(x, y) {
			a.cmdMenuOpen = false
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
	// The [Commands] button sits immediately right of [Menu] when present.
	if a.hasCmdMenu() {
		cmdStart := len(menuButtonLabel)
		if x >= cmdStart && x < cmdStart+len(cmdButtonLabel) {
			a.menuOpen = false
			a.cmdMenuOpen = !a.cmdMenuOpen
			if a.cmdMenuOpen {
				a.cmdPath = nil
				a.buildCmdMenu()
			}
			return
		}
	}
	// The [Menu] button (or any other spot on the bar) toggles the main menu.
	a.cmdMenuOpen = false
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

// handleCmdDropdownClick routes a click inside the command drop-down. It returns
// true if the click landed on the menu (a back row, a submenu, or a command).
func (a *App) handleCmdDropdownClick(x, y int) bool {
	cmdStart := len(menuButtonLabel)
	idx := y - 1
	if x < cmdStart || x >= cmdStart+a.cmdMenuWidth || idx < 0 {
		return false
	}
	entries := a.cmdCurrentEntries()
	if len(a.cmdPath) > 0 {
		if idx == 0 {
			// ".." back row: ascend one level.
			a.cmdPath = a.cmdPath[:len(a.cmdPath)-1]
			a.buildCmdMenu()
			return true
		}
		idx--
	}
	if idx >= len(entries) {
		return false
	}
	e := entries[idx]
	if len(e.Submenu) > 0 {
		// Branch: drill into the submenu.
		a.cmdPath = append(a.cmdPath, idx)
		a.buildCmdMenu()
		return true
	}
	// Leaf: run the command in a new pane.
	a.cmdMenuOpen = false
	a.runCommandInNewPane(e.Command, e.CloseAfter)
	return true
}
