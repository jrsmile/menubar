package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNested(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "menu.toml")
	content := `
[[item]]
label = "Build"
command = "make build"

[[item]]
label = "Tests"
command = "go test ./..."
close_after = true

[[item]]
label = "Git"

  [[item.submenu]]
  label = "Status"
  command = "git status"

  [[item.submenu]]
  label = "Branch"

    [[item.submenu.submenu]]
    label = "List"
    command = "git branch"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	items, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d top-level items, want 3", len(items))
	}

	if items[0].Label != "Build" || items[0].Command != "make build" {
		t.Errorf("item[0] = %+v", items[0])
	}
	if !items[1].CloseAfter {
		t.Errorf("item[1].CloseAfter = false, want true")
	}

	git := items[2]
	if git.Label != "Git" || len(git.Submenu) != 2 {
		t.Fatalf("git submenu = %+v", git.Submenu)
	}
	if git.Submenu[0].Command != "git status" {
		t.Errorf("git.Submenu[0] = %+v", git.Submenu[0])
	}

	branch := git.Submenu[1]
	if len(branch.Submenu) != 1 || branch.Submenu[0].Command != "git branch" {
		t.Errorf("nested submenu = %+v", branch.Submenu)
	}
}

func TestLoadMissingFileIsNotError(t *testing.T) {
	items, err := Load(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	if err != nil {
		t.Fatalf("Load of missing file returned error: %v", err)
	}
	if items != nil {
		t.Fatalf("got %v, want nil", items)
	}
}
