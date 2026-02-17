//go:build !windows

// Package pty manages pseudo-terminal spawning and I/O bridging
// between a PTY process and a WebSocket connection.
package pty

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// ResizeMsg is sent by xterm.js to resize the PTY.
// Wire format: first byte 0x01, followed by JSON {"cols":N,"rows":N}.
type ResizeMsg struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// Session represents a single PTY session.
type Session struct {
	Pty  *os.File
	Cmd  *exec.Cmd
	mu   sync.Mutex
	done chan struct{}
}

// Spawn starts a new PTY running the given command.
// Returns a Session that must be closed when done.
func Spawn(name string, args ...string) (*Session, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	// Set initial size.
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	return &Session{
		Pty:  ptmx,
		Cmd:  cmd,
		done: make(chan struct{}),
	}, nil
}

// Resize changes the PTY window size.
func (s *Session) Resize(cols, rows uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = pty.Setsize(s.Pty, &pty.Winsize{Rows: rows, Cols: cols})
}

// Read reads from the PTY (blocking).
func (s *Session) Read(buf []byte) (int, error) {
	return s.Pty.Read(buf)
}

// Write writes to the PTY stdin.
func (s *Session) Write(data []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Pty.Write(data)
}

// Close terminates the PTY session and cleans up.
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.done:
		return // already closed
	default:
	}
	close(s.done)

	// Kill the process group.
	if s.Cmd.Process != nil {
		_ = syscall.Kill(-s.Cmd.Process.Pid, syscall.SIGKILL)
	}
	_ = s.Pty.Close()

	// Wait to reap the child.
	_ = s.Cmd.Wait()
}

// Done returns a channel that is closed when the session ends.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

// HandleInput processes input from the WebSocket.
// Byte 0x01 prefix = resize message (JSON), otherwise raw PTY input.
func (s *Session) HandleInput(data []byte) {
	if len(data) == 0 {
		return
	}

	// Resize message: 0x01 + JSON.
	if data[0] == 0x01 && len(data) > 1 {
		var msg ResizeMsg
		if err := json.Unmarshal(data[1:], &msg); err == nil && msg.Cols > 0 && msg.Rows > 0 {
			s.Resize(msg.Cols, msg.Rows)
		}
		return
	}

	// Regular input.
	_, _ = s.Write(data)
}

// StreamOutput reads from the PTY and calls send for each chunk.
// Blocks until the PTY closes or an error occurs.
func (s *Session) StreamOutput(send func([]byte) error) {
	buf := make([]byte, 4096)
	for {
		n, err := s.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			if sendErr := send(chunk); sendErr != nil {
				log.Printf("pty: send error: %v", sendErr)
				return
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("pty: read error: %v", err)
			}
			return
		}
	}
}
