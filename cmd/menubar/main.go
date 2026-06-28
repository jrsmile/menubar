// Command menubar is a menu-bar terminal multiplexer: a single visible pane
// with a clickable top menu for creating, switching, and closing panes.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"

	"menubar/internal/config"
	"menubar/internal/mux"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	var (
		showVersion bool
		configPath  string
	)
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&showVersion, "v", false, "print version and exit (shorthand)")
	flag.StringVar(&configPath, "config", "", "path to the command-menu TOML file")
	flag.StringVar(&configPath, "c", "", "path to the command-menu TOML file (shorthand)")
	flag.Parse()

	if showVersion {
		fmt.Println("menubar", version)
		return
	}

	// Load the command menu. An explicit --config that fails is fatal; the
	// default path is best-effort (missing = no menu, invalid = warn + continue).
	explicit := configPath != ""
	if configPath == "" {
		configPath = config.DefaultPath()
	}
	cmdMenu, err := config.Load(configPath)
	if err != nil {
		if explicit {
			fmt.Fprintln(os.Stderr, "menubar:", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "menubar: ignoring %s: %v\n", configPath, err)
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, "menubar:", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "menubar:", err)
		os.Exit(1)
	}
	// Only button presses are needed; skipping motion/drag reporting keeps
	// PuTTY's Shift+drag native selection clean and avoids stray menu toggles.
	screen.EnableMouse(tcell.MouseButtonEvents)

	app := mux.New(screen, shell, cmdMenu)
	app.Run()

	screen.DisableMouse()
	screen.Fini()
}
