package mux

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hinshun/vt10x"
)

// draw repaints the whole screen: menu bar on row 0, active pane below, and the
// drop-down menu on top when open.
func (a *App) draw() {
	a.screen.Clear()
	w, h := a.screen.Size()

	a.drawPane(w, h)
	a.drawMenuBar(w)
	if a.menuOpen {
		a.drawDropdown()
	}
	if a.cmdMenuOpen {
		a.drawCmdDropdown()
	}
	if len(a.popups) > 0 {
		a.drawPopup(a.popups[0])
	}

	a.screen.Show()
}

// drawPane renders the active pane's emulator buffer into rows 1..h-1.
func (a *App) drawPane(w, h int) {
	if len(a.panes) == 0 {
		return
	}
	p := a.panes[a.active]
	rows := h - 1
	term := p.Term()

	term.Lock()
	cols, trows := term.Size()
	for y := 0; y < rows && y < trows; y++ {
		for x := 0; x < w && x < cols; x++ {
			g := term.Cell(x, y)
			ch := g.Char
			if ch == 0 {
				ch = ' '
			}
			a.screen.SetContent(x, y+1, ch, nil, glyphStyle(g))
		}
	}
	cur := term.Cursor()
	visible := term.CursorVisible()
	term.Unlock()

	if visible && !a.menuOpen && !a.cmdMenuOpen && len(a.popups) == 0 {
		a.screen.ShowCursor(cur.X, cur.Y+1)
	} else {
		a.screen.HideCursor()
	}
}

// drawMenuBar paints the top row: a Menu button on the left, a title in the
// middle, a clock, and a close (X) button on the right.
func (a *App) drawMenuBar(w int) {
	bar := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite)
	for x := 0; x < w; x++ {
		a.screen.SetContent(x, 0, ' ', nil, bar)
	}

	drawText(a.screen, 0, 0, menuButtonLabel, bar.Bold(true))

	leftEnd := len(menuButtonLabel)
	if a.hasCmdMenu() {
		drawText(a.screen, leftEnd, 0, cmdButtonLabel, bar.Bold(true))
		leftEnd += len(cmdButtonLabel)
	}

	clock := time.Now().Format("15:04:05")
	clockStart := w - len(xButtonLabel) - 1 - len(clock)

	title := a.title()
	maxTitle := clockStart - 1 - (leftEnd + 1)
	if maxTitle > 0 {
		if len(title) > maxTitle {
			title = title[:maxTitle]
		}
		drawText(a.screen, leftEnd+1, 0, title, bar)
	}

	if clockStart > leftEnd {
		drawText(a.screen, clockStart, 0, clock, bar)
	}

	xStyle := tcell.StyleDefault.Background(tcell.ColorMaroon).Foreground(tcell.ColorWhite).Bold(true)
	drawText(a.screen, w-len(xButtonLabel), 0, xButtonLabel, xStyle)
}

// drawDropdown paints the open menu starting just below the bar.
func (a *App) drawDropdown() {
	normal := tcell.StyleDefault.Background(tcell.ColorSilver).Foreground(tcell.ColorBlack)
	active := tcell.StyleDefault.Background(tcell.ColorTeal).Foreground(tcell.ColorWhite)

	// Menu layout is: [0]=New Pane, [1..n]=pane switchers, [last]=Close Pane,
	// so the active pane's switcher entry is at index 1+active.
	for i, it := range a.menuItems {
		style := normal
		if i == a.active+1 {
			style = active
		}
		row := make([]rune, a.menuWidth)
		for j := range row {
			row[j] = ' '
		}
		copy(row, []rune(" "+it.label))
		for j, r := range row {
			a.screen.SetContent(j, i+1, r, nil, style)
		}
	}
}

// drawCmdDropdown paints the user-defined command menu below the [Commands]
// button. It shows the current drill-down level: a ".." back row when nested,
// each entry's label, and a ">" marker on entries that open a submenu.
func (a *App) drawCmdDropdown() {
	normal := tcell.StyleDefault.Background(tcell.ColorSilver).Foreground(tcell.ColorBlack)
	branch := tcell.StyleDefault.Background(tcell.ColorSilver).Foreground(tcell.ColorNavy).Bold(true)

	xOff := len(menuButtonLabel)

	type cmdRow struct {
		label string
		style tcell.Style
	}
	var rows []cmdRow
	if len(a.cmdPath) > 0 {
		rows = append(rows, cmdRow{label: cmdBackLabel, style: normal})
	}
	for _, e := range a.cmdCurrentEntries() {
		label := e.Label
		style := normal
		if len(e.Submenu) > 0 {
			label += " >"
			style = branch
		}
		rows = append(rows, cmdRow{label: label, style: style})
	}

	for i, r := range rows {
		row := make([]rune, a.cmdMenuWidth)
		for j := range row {
			row[j] = ' '
		}
		copy(row, []rune(" "+r.label))
		for j, ch := range row {
			a.screen.SetContent(xOff+j, i+1, ch, nil, r.style)
		}
	}
}

func drawText(s tcell.Screen, x, y int, text string, style tcell.Style) {
	for _, r := range text {
		s.SetContent(x, y, r, nil, style)
		x++
	}
}

// drawPopup paints a centered modal box containing text and an OK button along
// the bottom. It records the OK button's hit-box in popupOK* for click matching.
func (a *App) drawPopup(text string) {
	w, h := a.screen.Size()

	maxW := w - 4
	if maxW < 10 {
		maxW = 10
	}
	maxLines := h - 6
	if maxLines < 1 {
		maxLines = 1
	}

	var lines []string
	for _, ln := range strings.Split(text, "\n") {
		lines = append(lines, wrapLine(ln, maxW)...)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines[len(lines)-1] = "…"
	}

	const okLabel = "[ OK ]"
	contentW := len(okLabel)
	for _, ln := range lines {
		if l := runeLen(ln); l > contentW {
			contentW = l
		}
	}

	boxW := contentW + 4 // 1-col border + 1-col padding on each side
	boxH := len(lines) + 4
	if boxW > w {
		boxW = w
	}
	if boxH > h {
		boxH = h
	}

	x0 := (w - boxW) / 2
	y0 := (h - boxH) / 2

	box := tcell.StyleDefault.Background(tcell.ColorSilver).Foreground(tcell.ColorBlack)
	okStyle := tcell.StyleDefault.Background(tcell.ColorTeal).Foreground(tcell.ColorWhite).Bold(true)

	for y := 0; y < boxH; y++ {
		for x := 0; x < boxW; x++ {
			ch := ' '
			switch {
			case y == 0 && x == 0:
				ch = '┌'
			case y == 0 && x == boxW-1:
				ch = '┐'
			case y == boxH-1 && x == 0:
				ch = '└'
			case y == boxH-1 && x == boxW-1:
				ch = '┘'
			case y == 0 || y == boxH-1:
				ch = '─'
			case x == 0 || x == boxW-1:
				ch = '│'
			}
			a.screen.SetContent(x0+x, y0+y, ch, nil, box)
		}
	}

	for i, ln := range lines {
		drawText(a.screen, x0+2, y0+1+i, ln, box)
	}

	okX := x0 + (boxW-len(okLabel))/2
	okY := y0 + boxH - 2
	drawText(a.screen, okX, okY, okLabel, okStyle)

	a.popupOKX = okX
	a.popupOKY = okY
	a.popupOKW = len(okLabel)
}

// wrapLine hard-wraps s into chunks of at most max runes, preserving an empty
// line for empty input.
func wrapLine(s string, max int) []string {
	if max < 1 {
		max = 1
	}
	r := []rune(s)
	if len(r) == 0 {
		return []string{""}
	}
	var out []string
	for len(r) > max {
		out = append(out, string(r[:max]))
		r = r[max:]
	}
	return append(out, string(r))
}

// runeLen returns the number of runes in s (its rendered column width for the
// plain text shown in popups).
func runeLen(s string) int { return len([]rune(s)) }

// glyphStyle translates a vt10x glyph into a tcell style. vt10x already bakes
// reverse-video (FG/BG swap) and bright-bold colours into the glyph, so we only
// carry over the remaining text attributes here.
func glyphStyle(g vt10x.Glyph) tcell.Style {
	style := tcell.StyleDefault.
		Foreground(toTColor(g.FG)).
		Background(toTColor(g.BG))

	if g.Mode&attrBold != 0 {
		style = style.Bold(true)
	}
	if g.Mode&attrUnderline != 0 {
		style = style.Underline(true)
	}
	if g.Mode&attrItalic != 0 {
		style = style.Italic(true)
	}
	if g.Mode&attrBlink != 0 {
		style = style.Blink(true)
	}
	if g.Mode&attrReverse != 0 {
		style = style.Reverse(true)
	}
	return style
}

// toTColor maps a vt10x colour to a tcell colour. Values below 256 are palette
// indices; higher values (below the Default* sentinels) are packed 24-bit RGB.
func toTColor(c vt10x.Color) tcell.Color {
	switch {
	case c == vt10x.DefaultFG || c == vt10x.DefaultBG || c == vt10x.DefaultCursor:
		return tcell.ColorDefault
	case c < 256:
		return tcell.PaletteColor(int(c))
	default:
		return tcell.NewRGBColor(int32((c>>16)&0xff), int32((c>>8)&0xff), int32(c&0xff))
	}
}
