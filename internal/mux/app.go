// Package mux implements the menu-bar terminal multiplexer: it owns the screen,
// the set of panes, the menu, and the event loop.
package mux

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hinshun/vt10x"

	"menubar/internal/config"
	"menubar/internal/input"
	"menubar/internal/pane"
)

// App holds the whole multiplexer state. All mutation happens on the main
// (event-loop) goroutine; pane pump goroutines only post events.
type App struct {
	screen tcell.Screen
	shell  string

	panes  []*pane.Pane
	active int
	nextID int

	menuOpen  bool
	menuItems []menuItem
	menuWidth int

	// cmdRoot holds the user-defined command menu loaded from TOML. cmdMenuOpen
	// tracks whether that drop-down is visible, and cmdPath is the drill-down
	// path (indices) into the currently displayed submenu level.
	cmdRoot      []config.MenuEntry
	cmdMenuOpen  bool
	cmdPath      []int
	cmdMenuWidth int

	// popups is a FIFO queue of modal messages; the first is shown until its OK
	// button is clicked. popupOK* records the OK button hit-box from the last
	// draw so clicks can be matched against it.
	popups   []string
	popupOKX int
	popupOKY int
	popupOKW int

	// notifyLn is the Unix socket that child `menubar --notify` processes connect
	// to; sockPath is its filesystem path (removed on shutdown).
	notifyLn net.Listener
	sockPath string

	dirty chan struct{}
	quit  bool
}

// New creates the multiplexer, starts its background tickers, and opens the
// first pane. The screen must already be initialized. cmdMenu holds the
// user-defined command menu (may be nil/empty to disable it).
func New(screen tcell.Screen, shell string, cmdMenu []config.MenuEntry) *App {
	a := &App{
		screen:  screen,
		shell:   shell,
		nextID:  1,
		cmdRoot: cmdMenu,
		dirty:   make(chan struct{}, 1),
	}
	// Coalescing forwarder: collapse bursts of pane output into redraw events.
	go func() {
		for range a.dirty {
			_ = a.screen.PostEvent(redrawEvent{})
			time.Sleep(10 * time.Millisecond)
		}
	}()
	// Tick once a second to keep the menu-bar clock current.
	go func() {
		t := time.NewTicker(time.Second)
		for range t.C {
			a.notifyRedraw()
		}
	}()

	a.startNotifyServer()
	a.newPane()
	return a
}

// startNotifyServer opens a per-process Unix socket and exports its path via the
// MENUBAR_SOCK environment variable, which panes inherit. Child processes run as
// `menubar --notify "text"` connect to it to request a popup. Failure to open
// the socket simply disables the notify feature.
func (a *App) startNotifyServer() {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		base = os.TempDir()
	}
	path := filepath.Join(base, fmt.Sprintf("menubar-%d.sock", os.Getpid()))
	_ = os.Remove(path)

	ln, err := net.Listen("unix", path)
	if err != nil {
		return
	}
	_ = os.Chmod(path, 0o600)

	a.notifyLn = ln
	a.sockPath = path
	_ = os.Setenv("MENUBAR_SOCK", path)

	go a.acceptNotify(ln)
}

// acceptNotify serves notify connections, posting each received message as a
// popup on the main event loop. Messages are size-limited and treated as plain
// text (never interpreted as commands or escape sequences).
func (a *App) acceptNotify(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
			data, _ := io.ReadAll(io.LimitReader(c, 64*1024))
			text := strings.TrimRight(string(data), "\n")
			if text != "" {
				_ = a.screen.PostEvent(notifyEvent{text: text})
			}
		}(conn)
	}
}

// notifyRedraw is the non-blocking signal pane goroutines call on new output.
func (a *App) notifyRedraw() {
	select {
	case a.dirty <- struct{}{}:
	default:
	}
}

func (a *App) paneSize() (cols, rows int) {
	w, h := a.screen.Size()
	rows = h - 1
	if rows < 1 {
		rows = 1
	}
	return w, rows
}

func (a *App) title() string {
	if len(a.panes) == 0 {
		return ""
	}
	return fmt.Sprintf("Pane %d/%d  (%s)", a.active+1, len(a.panes), a.shell)
}

// newPane creates a shell pane and makes it active.
func (a *App) newPane() {
	a.spawnPane("")
}

// spawnPane creates a shell pane (optionally rooted at dir) and makes it active.
func (a *App) spawnPane(dir string) {
	cols, rows := a.paneSize()
	p, err := pane.New(a.nextID, a.shell, dir, cols, rows)
	if err != nil {
		return
	}
	a.nextID++
	a.panes = append(a.panes, p)
	a.active = len(a.panes) - 1

	id := p.ID()
	go p.Pump(a.notifyRedraw, func() {
		_ = a.screen.PostEvent(paneExitEvent{id: id})
	})
}

// runCommandInNewPane opens a new pane rooted at the visible pane's working
// directory and runs command in it. When closeAfter is true the pane closes
// automatically once the command finishes; otherwise the shell stays
// interactive so the output remains visible until closed manually.
func (a *App) runCommandInNewPane(command string, closeAfter bool) {
	dir := ""
	if len(a.panes) > 0 {
		dir = a.panes[a.active].Cwd()
	}
	a.spawnPane(dir)
	if command == "" || len(a.panes) == 0 {
		return
	}
	suffix := "\n"
	if closeAfter {
		suffix = "\nexit\n"
	}
	a.panes[a.active].Write([]byte(command + suffix))
}

// runCommandPopup runs command in the background (via the shell) rooted at the
// visible pane's working directory and posts its combined output as a popup. It
// does not open a pane.
func (a *App) runCommandPopup(command string) {
	if command == "" {
		return
	}
	dir := ""
	if len(a.panes) > 0 {
		dir = a.panes[a.active].Cwd()
	}
	go func() {
		cmd := exec.Command(a.shell, "-c", command)
		cmd.Dir = dir
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		text := string(out)
		if err != nil {
			if text != "" && !strings.HasSuffix(text, "\n") {
				text += "\n"
			}
			text += "[" + err.Error() + "]"
		}
		if strings.TrimSpace(text) == "" {
			text = "(no output)"
		}
		_ = a.screen.PostEvent(notifyEvent{text: text})
	}()
}

// dismissPopup removes the currently displayed popup, revealing the next queued
// one if any.
func (a *App) dismissPopup() {
	if len(a.popups) > 0 {
		a.popups = a.popups[1:]
	}
}

func (a *App) setActive(i int) {
	if i < 0 || i >= len(a.panes) {
		return
	}
	a.active = i
}

func (a *App) closeActivePane() {
	if len(a.panes) == 0 {
		return
	}
	a.closePane(a.active)
}

func (a *App) removePaneByID(id int) {
	for i, p := range a.panes {
		if p.ID() == id {
			a.closePane(i)
			return
		}
	}
}

func (a *App) closePane(idx int) {
	if idx < 0 || idx >= len(a.panes) {
		return
	}
	a.panes[idx].Close()
	a.panes = append(a.panes[:idx], a.panes[idx+1:]...)

	if len(a.panes) == 0 {
		a.quit = true
		return
	}
	if a.active >= len(a.panes) {
		a.active = len(a.panes) - 1
	}
}

// resize keeps every pane's emulator and PTY in sync with the window.
func (a *App) resize() {
	a.screen.Sync()
	cols, rows := a.paneSize()
	for _, p := range a.panes {
		p.Resize(cols, rows)
	}
}

func (a *App) handleKey(ev *tcell.EventKey) {
	if len(a.popups) > 0 {
		// A popup is modal: swallow keys until its OK button is clicked.
		return
	}
	if a.menuOpen || a.cmdMenuOpen {
		if ev.Key() == tcell.KeyEsc {
			a.menuOpen = false
			a.cmdMenuOpen = false
		}
		return
	}
	if len(a.panes) == 0 {
		return
	}
	p := a.panes[a.active]
	appCursor := p.Term().Mode()&vt10x.ModeAppCursor != 0
	if b := input.Encode(ev, appCursor); b != nil {
		p.Write(b)
	}
}

// Run drives the event loop until the last pane closes, then releases panes.
func (a *App) Run() {
	defer a.closeAll()

	a.draw()
	for !a.quit {
		ev := a.screen.PollEvent()
		if ev == nil {
			return
		}
		switch ev := ev.(type) {
		case *tcell.EventResize:
			a.resize()
		case *tcell.EventKey:
			a.handleKey(ev)
		case *tcell.EventMouse:
			a.handleMouse(ev)
		case paneExitEvent:
			a.removePaneByID(ev.id)
		case notifyEvent:
			a.popups = append(a.popups, ev.text)
		case redrawEvent:
			// fall through to repaint
		}
		a.draw()
	}
}

func (a *App) closeAll() {
	if a.notifyLn != nil {
		_ = a.notifyLn.Close()
	}
	if a.sockPath != "" {
		_ = os.Remove(a.sockPath)
	}
	for _, p := range a.panes {
		p.Close()
	}
}
