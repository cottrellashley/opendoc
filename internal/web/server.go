package web

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	ptyPkg "github.com/cottrellashley/opendoc/internal/pty"
	"github.com/gorilla/websocket"
)

// Config holds server configuration.
type Config struct {
	Port        int
	MaxSessions int
	IdleTimeout time.Duration
	TUIBinary   string // path to the TUI binary (opendoc console tui)
}

// Server is the HTTP/WebSocket console server.
type Server struct {
	config   Config
	sessions *SessionManager
	upgrader websocket.Upgrader
}

// NewServer creates a new console server.
func NewServer(cfg Config) *Server {
	return &Server{
		config:   cfg,
		sessions: NewSessionManager(cfg.MaxSessions, cfg.IdleTimeout),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins (container-internal).
			},
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
		},
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.HandleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.config.Port)
	log.Printf("console: listening on %s (max sessions: %d)", addr, s.config.MaxSessions)
	return http.ListenAndServe(addr, mux)
}

// handleHealth returns a simple health check response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","sessions":%d,"max":%d}`, s.sessions.Count(), s.config.MaxSessions)
}

// HandleWebSocket upgrades to WebSocket and spawns a PTY session.
// Exported so it can be mounted on external routers (e.g. workbench).
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !s.sessions.CanAccept() {
		http.Error(w, "Server busy: max sessions reached", http.StatusServiceUnavailable)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("console: websocket upgrade error: %v", err)
		return
	}

	sessionID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())
	log.Printf("console: new session %s", sessionID)

	// Determine session mode from query parameter.
	// ?mode=shell spawns a real bash shell; default spawns the TUI.
	mode := r.URL.Query().Get("mode")

	var sess *ptyPkg.Session
	if mode == "shell" {
		log.Printf("console: session %s mode=shell", sessionID)
		sess, err = ptyPkg.Spawn("/bin/bash", "--login")
	} else {
		tuiBinary := s.config.TUIBinary
		if tuiBinary == "" {
			tuiBinary, _ = os.Executable()
		}
		log.Printf("console: session %s mode=tui", sessionID)
		sess, err = ptyPkg.Spawn(tuiBinary, "console", "tui")
	}
	if err != nil {
		log.Printf("console: pty spawn error: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("Error: failed to start console\r\n"))
		conn.Close()
		return
	}

	if !s.sessions.Add(sessionID, sess) {
		sess.Close()
		conn.WriteMessage(websocket.TextMessage, []byte("Error: max sessions reached\r\n"))
		conn.Close()
		return
	}

	// Cleanup on exit.
	defer func() {
		s.sessions.Remove(sessionID)
		conn.Close()
		log.Printf("console: session %s closed", sessionID)
	}()

	// PTY output → WebSocket.
	go func() {
		sess.StreamOutput(func(data []byte) error {
			return conn.WriteMessage(websocket.BinaryMessage, data)
		})
		conn.Close()
	}()

	// WebSocket input → PTY.
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		s.sessions.Touch(sessionID)
		sess.HandleInput(msg)
	}
}
