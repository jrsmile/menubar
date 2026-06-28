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
- **Command menu** — an optional `[Commands]` button driven by a TOML file, with
  arbitrarily nested submenus; clicking an entry runs its command in a new pane
  rooted at the visible shell's working directory.
- **Popups** — show a modal message box from inside a pane with
  `menubar --notify "text"`, or run a menu command in the background and display
  its output in a popup (`popup = true`). Popups stay up until their `[ OK ]`
  button is clicked.
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
| Open command menu | Click `[Commands]` (shown only when configured)    |
| Run a command     | Commands → entry; `..` ascends, `>` opens a submenu |
| Dismiss a popup   | Click the `[ OK ]` button inside the popup          |
| Quit              | Close the last remaining pane                      |
| Select / paste    | Use your terminal's native Shift+click / paste     |

## Command menu

The `[Commands]` button appears when a command-menu TOML file is found. By
default it is read from `~/.config/menubar/menu.toml`; pass `--config`/`-c` to
point at a different file:

```sh
./menubar --config ./menu.toml
```

Each `[[item]]` is either a runnable leaf (with a `command`) or a branch (with a
`submenu`). Submenus may nest to any depth. Clicking a leaf opens a new pane in
the **current working directory of the visible shell** and runs the command.

```toml
[[item]]
label = "Build"
command = "make build"          # stays open; output visible until closed manually

[[item]]
label = "Run tests"
command = "go test ./..."
close_after = true              # pane closes automatically once the command exits

[[item]]
label = "Disk usage"
command = "df -h"
popup = true                    # runs in the background; output shown in a popup

[[item]]
label = "Git"
  [[item.submenu]]
  label = "Status"
  command = "git status"
  [[item.submenu]]
  label = "Log"
  command = "git log --oneline"
```

Per-entry fields:

| Field         | Meaning                                                          |
| ------------- | ---------------------------------------------------------------- |
| `label`       | Text shown in the menu.                                          |
| `command`     | Shell command to run (leaf entries).                            |
| `close_after` | `true` closes the pane when the command finishes; default keeps it open. |
| `popup`       | `true` runs the command in the background and shows its output in a popup instead of a pane. |
| `submenu`     | Nested `[[item.submenu]]` entries (branch entries).             |

## Popups & notifications

Popups are modal message boxes drawn over the screen; each stays visible until
its `[ OK ]` button is clicked.

From inside any pane you can raise a popup with the `--notify` flag:

```sh
menubar --notify "build finished"
```

This works because each pane inherits a `MENUBAR_SOCK` environment variable
pointing at the parent process's control socket; `--notify` simply sends the
text there. Running `menubar --notify` outside a menubar pane (no
`MENUBAR_SOCK`) prints an error and exits non-zero.

Menu entries with `popup = true` (see above) run their command in the background
and display the combined output in a popup instead of opening a pane.

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
