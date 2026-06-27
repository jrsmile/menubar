// Package input translates tcell key events into the byte sequences a terminal
// application expects on its stdin, for forwarding to a pane's PTY.
package input

import "github.com/gdamore/tcell/v2"

// Encode converts a tcell key event into the bytes a terminal application
// expects on its stdin. appCursor selects between normal (ESC [) and application
// (ESC O) cursor key encodings, mirroring the emulator's DECCKM mode. It returns
// nil for events that should not be forwarded.
func Encode(ev *tcell.EventKey, appCursor bool) []byte {
	mod := ev.Modifiers()

	switch ev.Key() {
	case tcell.KeyRune:
		var b []byte
		if r := ev.Rune(); mod&tcell.ModCtrl != 0 && r < 0x80 {
			// Ctrl reported alongside a rune: fold to the control byte.
			b = []byte{byte(r) & 0x1f}
		} else {
			b = []byte(string(r))
		}
		if mod&tcell.ModAlt != 0 {
			return append([]byte{0x1b}, b...)
		}
		return b
	case tcell.KeyEnter:
		return []byte{'\r'}
	case tcell.KeyTab:
		return []byte{'\t'}
	case tcell.KeyBacktab:
		return []byte("\x1b[Z")
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return []byte{0x7f}
	case tcell.KeyEsc:
		return []byte{0x1b}
	case tcell.KeyUp:
		return cursorSeq('A', appCursor)
	case tcell.KeyDown:
		return cursorSeq('B', appCursor)
	case tcell.KeyRight:
		return cursorSeq('C', appCursor)
	case tcell.KeyLeft:
		return cursorSeq('D', appCursor)
	case tcell.KeyHome:
		return cursorSeq('H', appCursor)
	case tcell.KeyEnd:
		return cursorSeq('F', appCursor)
	case tcell.KeyPgUp:
		return []byte("\x1b[5~")
	case tcell.KeyPgDn:
		return []byte("\x1b[6~")
	case tcell.KeyInsert:
		return []byte("\x1b[2~")
	case tcell.KeyDelete:
		return []byte("\x1b[3~")
	case tcell.KeyF1:
		return []byte("\x1bOP")
	case tcell.KeyF2:
		return []byte("\x1bOQ")
	case tcell.KeyF3:
		return []byte("\x1bOR")
	case tcell.KeyF4:
		return []byte("\x1bOS")
	case tcell.KeyF5:
		return []byte("\x1b[15~")
	case tcell.KeyF6:
		return []byte("\x1b[17~")
	case tcell.KeyF7:
		return []byte("\x1b[18~")
	case tcell.KeyF8:
		return []byte("\x1b[19~")
	case tcell.KeyF9:
		return []byte("\x1b[20~")
	case tcell.KeyF10:
		return []byte("\x1b[21~")
	case tcell.KeyF11:
		return []byte("\x1b[23~")
	case tcell.KeyF12:
		return []byte("\x1b[24~")
	}

	// tcell encodes Ctrl-<key> in the range [KeyCtrlSpace, KeyCtrlUnderscore]
	// (64..95), which maps directly to ASCII control bytes 0..31 (so Ctrl-C ->
	// 0x03). This is the path real terminals take for Ctrl combinations.
	if k := ev.Key(); k >= tcell.KeyCtrlSpace && k <= tcell.KeyCtrlUnderscore {
		b := []byte{byte(k - tcell.KeyCtrlSpace)}
		if mod&tcell.ModAlt != 0 {
			return append([]byte{0x1b}, b...)
		}
		return b
	}

	// Any remaining low-ASCII control keys map to their byte value.
	if k := ev.Key(); k > 0 && k < 0x20 {
		return []byte{byte(k)}
	}

	return nil
}

func cursorSeq(final byte, appCursor bool) []byte {
	if appCursor {
		return []byte{0x1b, 'O', final}
	}
	return []byte{0x1b, '[', final}
}
