package mux

import "time"

// redrawEvent asks the main loop to repaint after pane output changed.
type redrawEvent struct{}

func (redrawEvent) When() time.Time { return time.Time{} }

// paneExitEvent reports that a pane's shell has terminated.
type paneExitEvent struct{ id int }

func (paneExitEvent) When() time.Time { return time.Time{} }

// notifyEvent asks the main loop to display text in a modal popup. It is posted
// by the notify socket server and by background popup commands.
type notifyEvent struct{ text string }

func (notifyEvent) When() time.Time { return time.Time{} }
