package input

import (
	"bytes"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestEncodeCtrlKeys(t *testing.T) {
	cases := []struct {
		name string
		key  tcell.Key
		want []byte
	}{
		{"Ctrl-C", tcell.KeyCtrlC, []byte{0x03}},
		{"Ctrl-A", tcell.KeyCtrlA, []byte{0x01}},
		{"Ctrl-D", tcell.KeyCtrlD, []byte{0x04}},
		{"Ctrl-Z", tcell.KeyCtrlZ, []byte{0x1a}},
		{"Ctrl-Space", tcell.KeyCtrlSpace, []byte{0x00}},
		{"Ctrl-Underscore", tcell.KeyCtrlUnderscore, []byte{0x1f}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ev := tcell.NewEventKey(c.key, 0, tcell.ModCtrl)
			got := Encode(ev, false)
			if !bytes.Equal(got, c.want) {
				t.Fatalf("%s: got %v, want %v", c.name, got, c.want)
			}
		})
	}
}

func TestEncodeCtrlRune(t *testing.T) {
	// Some terminals report Ctrl-C as a rune 'c' with the Ctrl modifier.
	ev := tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModCtrl)
	if got := Encode(ev, false); !bytes.Equal(got, []byte{0x03}) {
		t.Fatalf("Ctrl-rune c: got %v", got)
	}
}

func TestEncodePlainRune(t *testing.T) {
	ev := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
	if got := Encode(ev, false); !bytes.Equal(got, []byte("a")) {
		t.Fatalf("plain rune: got %v", got)
	}
}

func TestEncodeArrowsAppCursor(t *testing.T) {
	ev := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	if got := Encode(ev, false); !bytes.Equal(got, []byte("\x1b[A")) {
		t.Fatalf("normal up: got %q", got)
	}
	if got := Encode(ev, true); !bytes.Equal(got, []byte("\x1bOA")) {
		t.Fatalf("appcursor up: got %q", got)
	}
}
