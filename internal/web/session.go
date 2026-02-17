// Package web implements the HTTP/WebSocket server for the OpenDoc console.
package web

import (
	"log"
	"sync"
	"time"

	ptyPkg "github.com/cottrellashley/opendoc/internal/pty"
)

// sessionEntry wraps a PTY session with idle-tracking metadata.
type sessionEntry struct {
	session      *ptyPkg.Session
	lastActivity time.Time
}

// SessionManager tracks active PTY sessions and enforces limits.
type SessionManager struct {
	mu          sync.Mutex
	sessions    map[string]*sessionEntry
	maxSessions int
	idleTimeout time.Duration
}

// NewSessionManager creates a session manager with the given limits.
func NewSessionManager(maxSessions int, idleTimeout time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions:    make(map[string]*sessionEntry),
		maxSessions: maxSessions,
		idleTimeout: idleTimeout,
	}
	go sm.reaper()
	return sm
}

// Count returns the number of active sessions.
func (sm *SessionManager) Count() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return len(sm.sessions)
}

// CanAccept returns true if a new session can be created.
func (sm *SessionManager) CanAccept() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return len(sm.sessions) < sm.maxSessions
}

// Add registers a new session. Returns false if the limit is reached.
func (sm *SessionManager) Add(id string, s *ptyPkg.Session) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if len(sm.sessions) >= sm.maxSessions {
		return false
	}
	sm.sessions[id] = &sessionEntry{
		session:      s,
		lastActivity: time.Now(),
	}
	return true
}

// Touch updates the last-activity timestamp for a session.
func (sm *SessionManager) Touch(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if entry, ok := sm.sessions[id]; ok {
		entry.lastActivity = time.Now()
	}
}

// Remove removes and closes a session.
func (sm *SessionManager) Remove(id string) {
	sm.mu.Lock()
	entry, ok := sm.sessions[id]
	if ok {
		delete(sm.sessions, id)
	}
	sm.mu.Unlock()

	if ok && entry != nil {
		entry.session.Close()
	}
}

// GetSession returns the PTY session for the given ID, or nil.
func (sm *SessionManager) GetSession(id string) *ptyPkg.Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if entry, ok := sm.sessions[id]; ok {
		return entry.session
	}
	return nil
}

// reaper periodically cleans up dead and idle sessions.
func (sm *SessionManager) reaper() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for id, entry := range sm.sessions {
			// Reap dead sessions.
			select {
			case <-entry.session.Done():
				log.Printf("session: reaping dead session %s", id)
				delete(sm.sessions, id)
				continue
			default:
			}

			// Reap idle sessions.
			if sm.idleTimeout > 0 && now.Sub(entry.lastActivity) > sm.idleTimeout {
				log.Printf("session: reaping idle session %s (idle %s)", id, now.Sub(entry.lastActivity))
				entry.session.Close()
				delete(sm.sessions, id)
			}
		}
		sm.mu.Unlock()
	}
}
