// Package config loads the user-defined command menu from a TOML file. Each
// entry is either a leaf (with a command to run) or a branch (with a submenu),
// and submenus may nest arbitrarily.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// MenuEntry is one entry in the command menu. An entry with a Submenu acts as a
// branch; an entry with a Command acts as a runnable leaf. CloseAfter controls
// whether the spawned pane closes once the command finishes. When Popup is true
// the command runs in the background and its output is shown in a popup instead
// of in a new pane.
type MenuEntry struct {
	Label      string      `toml:"label"`
	Command    string      `toml:"command"`
	CloseAfter bool        `toml:"close_after"`
	Popup      bool        `toml:"popup"`
	Submenu    []MenuEntry `toml:"submenu"`
}

// file is the top-level TOML document shape.
type file struct {
	Items []MenuEntry `toml:"item"`
}

// DefaultPath returns the standard location of the command-menu file
// (~/.config/menubar/menu.toml on Linux).
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "menubar", "menu.toml")
}

// Load reads and parses the command-menu file at path. It returns the top-level
// menu entries. A non-existent file is not an error: the returned slice is nil.
func Load(path string) ([]MenuEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var f file
	if err := toml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Items, nil
}
