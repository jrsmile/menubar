# menubar

A mouse-driven terminal multiplexer for the terminal. `menubar` shows a single
shell pane below a clickable menu bar. You create, switch between, and close
panes entirely from the top menu — there are no keyboard shortcuts to learn, so
every keystroke is forwarded straight to the active shell.

It is designed to behave well inside PuTTY: native Shift+click selection and
right-click paste keep working because the app only listens for plain mouse
button presses on its own menu bar.

## Features

- **Tabbed panes** — one visible shell at a time, switched via the menu.
- **Mouse-only controls** — a `[Menu]` button (left), the active pane title and
  a live clock (center/right), and an `[X]` close button (top-right).
- **Full key passthrough** — keys, including `Ctrl-C` and other control
  combinations, go to the active pane unchanged.
- **PuTTY-friendly** — only button presses are captured, leaving PuTTY's
  Shift+drag selection and right-click paste untouched.
- **Self-answering Device Attributes** — panes reply to terminal DA queries that
  the emulator leaves unanswered, so shells like `fish` don't stall on startup.

## Usage

Build and run:

```sh
make run
```

Or build the binary and launch it directly:

```sh
make build
./menubar
```

The shell launched in each pane is taken from `$SHELL` (falling back to
`/bin/sh`).

### Controls

| Action            | How                                                |
| ----------------- | -------------------------------------------------- |
| Open/close menu   | Click `[Menu]` on the top-left                     |
| New pane          | Menu → **New Pane**                                |
| Switch pane       | Menu → **Go to Pane N** (the active one is marked) |
| Close active pane | Menu → **Close Pane**, or click `[X]` top-right    |
| Quit              | Close the last remaining pane                      |
| Select / paste    | Use your terminal's native Shift+click / paste     |

## Architecture

The project follows the standard Go layout, with a thin entrypoint under
`cmd/` and the implementation split into focused packages under `internal/`.

```
cmd/menubar/main.go     Entrypoint: read $SHELL, init tcell, run the app.
internal/pane/          A single shell: PTY + vt10x emulator + DA responder.
internal/input/         Pure key encoder: tcell key events -> PTY byte sequences.
internal/mux/           The multiplexer: app state, event loop, rendering, menu.
```

- **`internal/pane`** owns one shell process inside a PTY and a `vt10x` terminal
  emulator over its output. It is intentionally decoupled from the UI: `Pump`
  takes `onData`/`onExit` callbacks instead of depending on `tcell`, so it can be
  tested in isolation.
- **`internal/input`** is a pure function, `Encode`, that translates a `tcell`
  key event into the bytes a terminal application expects on stdin (handling
  control keys, arrows in normal vs. application-cursor mode, function keys,
  etc.).
- **`internal/mux`** ties everything together: it holds the set of panes, runs
  the `tcell` event loop, draws the menu bar and active pane, and handles mouse
  interaction. Pane output is coalesced into redraw events, and a one-second
  ticker keeps the clock current.

### Rendering model

Row 0 is the menu bar; rows 1 and below show the active pane's emulator buffer.
Each pane runs a goroutine that pumps PTY output into its emulator and signals a
(coalesced) redraw. All multiplexer state is mutated only on the main event-loop
goroutine — pane goroutines merely post events.

## Development

Common tasks via the `Makefile`:

```sh
make build   # build ./cmd/menubar -> ./menubar
make run     # build and run
make test    # go test ./...
make vet     # go vet ./...
make fmt     # gofmt -w .
make tidy    # go mod tidy
make clean   # remove the binary
```

### Dependencies

- [`github.com/gdamore/tcell/v2`](https://github.com/gdamore/tcell) — screen
  rendering, keyboard, and mouse input.
- [`github.com/creack/pty`](https://github.com/creack/pty) — pseudo-terminal
  management.
- [`github.com/hinshun/vt10x`](https://github.com/hinshun/vt10x) — terminal
  emulation over each pane's output.
