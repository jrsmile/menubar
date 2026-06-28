package mux

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"menubar/internal/config"
)

// row0 returns the text rendered on the menu bar (screen row 0).
func row0(t *testing.T, sim tcell.SimulationScreen) string {
	t.Helper()
	cells, w, _ := sim.GetContents()
	var b strings.Builder
	for x := 0; x < w; x++ {
		if r := cells[x].Runes; len(r) > 0 {
			b.WriteRune(r[0])
		}
	}
	return b.String()
}

func TestMenuBarShowsCommandsButtonWhenConfigured(t *testing.T) {
	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	defer sim.Fini()
	sim.SetSize(80, 24)

	a := &App{
		screen:  sim,
		cmdRoot: []config.MenuEntry{{Label: "Run tests", Command: "go test ./..."}},
	}
	a.drawMenuBar(80)
	sim.Show()

	if got := row0(t, sim); !strings.Contains(got, cmdButtonLabel) {
		t.Errorf("menu bar = %q, want it to contain %q", got, cmdButtonLabel)
	}
}

func TestMenuBarHidesCommandsButtonWhenNoConfig(t *testing.T) {
	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	defer sim.Fini()
	sim.SetSize(80, 24)

	a := &App{screen: sim}
	a.drawMenuBar(80)
	sim.Show()

	if got := row0(t, sim); strings.Contains(got, cmdButtonLabel) {
		t.Errorf("menu bar = %q, should not contain %q", got, cmdButtonLabel)
	}
}
