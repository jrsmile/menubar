package mux

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func newSimApp(t *testing.T) (*App, tcell.SimulationScreen) {
	t.Helper()
	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(sim.Fini)
	sim.SetSize(80, 24)
	return &App{screen: sim}, sim
}

func screenText(sim tcell.SimulationScreen) string {
	cells, w, h := sim.GetContents()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if r := cells[y*w+x].Runes; len(r) > 0 {
				b.WriteRune(r[0])
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func TestPopupRendersAndDismissesOnOKClick(t *testing.T) {
	a, sim := newSimApp(t)
	a.popups = []string{"hello world"}

	a.draw()
	if got := screenText(sim); !strings.Contains(got, "hello world") || !strings.Contains(got, "[ OK ]") {
		t.Fatalf("popup not rendered:\n%s", got)
	}

	// Clicking away from OK must not dismiss it.
	a.handlePopupClick(0, 0)
	if len(a.popups) != 1 {
		t.Fatalf("popup dismissed by off-target click")
	}

	// Clicking OK dismisses it.
	a.handlePopupClick(a.popupOKX, a.popupOKY)
	if len(a.popups) != 0 {
		t.Fatalf("popup not dismissed by OK click; have %d", len(a.popups))
	}
}

func TestPopupQueueShowsNextAfterDismiss(t *testing.T) {
	a, _ := newSimApp(t)
	a.popups = []string{"first", "second"}
	a.draw()
	a.handlePopupClick(a.popupOKX, a.popupOKY)
	if len(a.popups) != 1 || a.popups[0] != "second" {
		t.Fatalf("expected [second], got %v", a.popups)
	}
}

func TestNotifySocketDeliversPopup(t *testing.T) {
	a, sim := newSimApp(t)
	a.startNotifyServer()
	if a.sockPath == "" {
		t.Skip("could not open notify socket")
	}
	t.Cleanup(func() {
		_ = a.notifyLn.Close()
		_ = os.Remove(a.sockPath)
	})

	conn, err := net.Dial("unix", os.Getenv("MENUBAR_SOCK"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write([]byte("from child\n")); err != nil {
		t.Fatal(err)
	}
	conn.Close()

	events := make(chan tcell.Event, 16)
	go func() {
		for {
			ev := sim.PollEvent()
			if ev == nil {
				return
			}
			events <- ev
		}
	}()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("did not receive notifyEvent")
		case ev := <-events:
			if ne, ok := ev.(notifyEvent); ok {
				if ne.text != "from child" {
					t.Fatalf("got %q, want %q", ne.text, "from child")
				}
				return
			}
		}
	}
}
