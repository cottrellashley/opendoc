//go:build windows

// Package pty provides stubs on Windows where PTY is not supported.
package pty

import (
	"errors"
	"os"
	"os/exec"
)

// ResizeMsg is sent by xterm.js to resize the PTY.
type ResizeMsg struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// Session represents a single PTY session (stub on Windows).
type Session struct {
	Pty  *os.File
	Cmd  *exec.Cmd
	done chan struct{}
}

var errNotSupported = errors.New("pty: not supported on Windows")

// Spawn is not available on Windows.
func Spawn(name string, args ...string) (*Session, error) {
	return nil, errNotSupported
}

// Resize is a no-op on Windows.
func (s *Session) Resize(cols, rows uint16) {}

// Read always returns an error on Windows.
func (s *Session) Read(buf []byte) (int, error) { return 0, errNotSupported }

// Write always returns an error on Windows.
func (s *Session) Write(data []byte) (int, error) { return 0, errNotSupported }

// Close is a no-op on Windows.
func (s *Session) Close() {}

// Done returns a closed channel on Windows.
func (s *Session) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// HandleInput is a no-op on Windows.
func (s *Session) HandleInput(data []byte) {}

// StreamOutput is a no-op on Windows.
func (s *Session) StreamOutput(send func([]byte) error) {}
