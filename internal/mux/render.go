package mux

import (
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

	if visible && !a.menuOpen {
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

	clock := time.Now().Format("15:04:05")
	clockStart := w - len(xButtonLabel) - 1 - len(clock)

	title := a.title()
	maxTitle := clockStart - 1 - (len(menuButtonLabel) + 1)
	if maxTitle > 0 {
		if len(title) > maxTitle {
			title = title[:maxTitle]
		}
		drawText(a.screen, len(menuButtonLabel)+1, 0, title, bar)
	}

	if clockStart > len(menuButtonLabel) {
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

func drawText(s tcell.Screen, x, y int, text string, style tcell.Style) {
	for _, r := range text {
		s.SetContent(x, y, r, nil, style)
		x++
	}
}

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
