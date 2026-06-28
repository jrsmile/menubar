// Command menubar is a menu-bar terminal multiplexer: a single visible pane
// with a clickable top menu for creating, switching, and closing panes.
package main

import (
	"flag"
	"fmt"
	"net"
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
		notifyText  string
	)
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&showVersion, "v", false, "print version and exit (shorthand)")
	flag.StringVar(&configPath, "config", "", "path to the command-menu TOML file")
	flag.StringVar(&configPath, "c", "", "path to the command-menu TOML file (shorthand)")
	flag.StringVar(&notifyText, "notify", "", "show text as a popup in the parent menubar (run inside a pane)")
	flag.Parse()

	if showVersion {
		fmt.Println("menubar", version)
		return
	}

	// --notify runs as a thin client: send the text to the parent menubar's
	// socket and exit, without starting a screen of our own.
	notifySet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "notify" {
			notifySet = true
		}
	})
	if notifySet {
		if err := sendNotify(notifyText); err != nil {
			fmt.Fprintln(os.Stderr, "menubar:", err)
			os.Exit(1)
		}
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

// sendNotify connects to the parent menubar's notify socket (advertised via the
// MENUBAR_SOCK environment variable) and asks it to display text in a popup.
func sendNotify(text string) error {
	sock := os.Getenv("MENUBAR_SOCK")
	if sock == "" {
		return fmt.Errorf("--notify must be run inside a menubar pane")
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte(text))
	return err
}
