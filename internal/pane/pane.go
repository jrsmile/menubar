// Package pane runs a single shell inside a PTY and maintains a terminal
// emulator (vt10x) over its output, exposing a renderable view.
package pane

import (
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

const readBufSize = 16 * 1024

// Pane is a single shell running inside its own PTY, with a vt10x emulator
// parsing the shell's output into a renderable screen buffer.
type Pane struct {
	id     int
	cmd    *exec.Cmd
	ptmx   *os.File
	term   vt10x.Terminal
	closed bool

	writeMu sync.Mutex // serializes writes to the PTY (keys + query replies)

	// Minimal CSI scanner state used to answer Device Attributes queries that
	// vt10x leaves unanswered (otherwise shells like fish stall for ~10s).
	qState  int
	qParams []byte
}

// New spawns a shell in a fresh PTY sized to cols x rows and attaches a vt10x
// terminal emulator to it.
func New(id int, shell string, cols, rows int) (*Pane, error) {
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
	if err != nil {
		return nil, err
	}

	term := vt10x.New(vt10x.WithWriter(ptmx), vt10x.WithSize(cols, rows))

	return &Pane{
		id:   id,
		cmd:  cmd,
		ptmx: ptmx,
		term: term,
	}, nil
}

// ID returns the pane's stable identifier.
func (p *Pane) ID() int { return p.id }

// Term exposes the emulator view for rendering and mode queries.
func (p *Pane) Term() vt10x.Terminal { return p.term }

// Pump continuously reads PTY output into the emulator. onData is invoked after
// each chunk so callers can schedule a redraw; onExit is invoked once when the
// shell terminates (read error/EOF).
func (p *Pane) Pump(onData, onExit func()) {
	buf := make([]byte, readBufSize)
	for {
		n, err := p.ptmx.Read(buf)
		if n > 0 {
			_, _ = p.term.Write(buf[:n])
			p.scanQueries(buf[:n])
			if onData != nil {
				onData()
			}
		}
		if err != nil {
			if onExit != nil {
				onExit()
			}
			return
		}
	}
}

// Write sends bytes to the PTY, serialized so concurrent key input and query
// replies never interleave.
func (p *Pane) Write(b []byte) {
	p.writeMu.Lock()
	_, _ = p.ptmx.Write(b)
	p.writeMu.Unlock()
}

// scanQueries inspects PTY output for Device Attributes (DA) queries and replies
// on the pane's behalf. vt10x parses but does not answer these, which makes some
// shells wait for a response that never comes. All bytes are still fed to vt10x
// unchanged; this scan is passive.
func (p *Pane) scanQueries(b []byte) {
	for _, c := range b {
		switch p.qState {
		case 0: // normal
			if c == 0x1b {
				p.qState = 1
			}
		case 1: // saw ESC
			if c == '[' {
				p.qState = 2
				p.qParams = p.qParams[:0]
			} else if c != 0x1b {
				p.qState = 0
			}
		case 2: // inside CSI, collecting params until a final byte
			if c >= 0x40 && c <= 0x7e {
				p.answerCSI(c)
				p.qState = 0
			} else if len(p.qParams) < 32 {
				p.qParams = append(p.qParams, c)
			}
		}
	}
}

// answerCSI replies to a completed CSI sequence if it is a DA query (final 'c').
func (p *Pane) answerCSI(final byte) {
	if final != 'c' {
		return
	}
	switch {
	case len(p.qParams) > 0 && p.qParams[0] == '>':
		// Secondary DA: report a generic VT220-class terminal.
		p.Write([]byte("\x1b[>0;0;0c"))
	case len(p.qParams) > 0 && p.qParams[0] == '=':
		// Tertiary DA: ignored (not needed by common shells).
	default:
		// Primary DA: report a VT102.
		p.Write([]byte("\x1b[?6c"))
	}
}

// Resize updates both the emulator and the PTY window size.
func (p *Pane) Resize(cols, rows int) {
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	p.term.Resize(cols, rows)
	_ = pty.Setsize(p.ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

// Close terminates the shell and releases the PTY. Safe to call more than once.
func (p *Pane) Close() {
	if p.closed {
		return
	}
	p.closed = true
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	_ = p.ptmx.Close()
	_ = p.cmd.Wait()
}
