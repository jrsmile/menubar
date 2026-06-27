package pane

import (
	"io"
	"os"
	"testing"
)

// readResponse runs scanQueries against input and returns whatever the pane
// wrote back to its PTY.
func readResponse(t *testing.T, chunks ...[]byte) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	p := &Pane{ptmx: w}
	for _, c := range chunks {
		p.scanQueries(c)
	}
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestPrimaryDA(t *testing.T) {
	if got := readResponse(t, []byte("\x1b[c")); got != "\x1b[?6c" {
		t.Fatalf("primary DA: got %q", got)
	}
}

func TestPrimaryDAWithZeroParam(t *testing.T) {
	if got := readResponse(t, []byte("\x1b[0c")); got != "\x1b[?6c" {
		t.Fatalf("primary DA (0): got %q", got)
	}
}

func TestSecondaryDA(t *testing.T) {
	if got := readResponse(t, []byte("\x1b[>c")); got != "\x1b[>0;0;0c" {
		t.Fatalf("secondary DA: got %q", got)
	}
}

func TestDASplitAcrossReads(t *testing.T) {
	if got := readResponse(t, []byte("ls\x1b["), []byte("c")); got != "\x1b[?6c" {
		t.Fatalf("split DA: got %q", got)
	}
}

func TestNonQueryCSIIgnored(t *testing.T) {
	if got := readResponse(t, []byte("\x1b[2J\x1b[?25h")); got != "" {
		t.Fatalf("expected no response, got %q", got)
	}
}
