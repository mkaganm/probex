package server

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/mkaganm/probex/internal/ai"
	"github.com/mkaganm/probex/internal/dashboard"
	"github.com/mkaganm/probex/internal/storage"
)

// Server provides a REST API for SDK clients (Java, JS/TS).
type Server struct {
	addr     string
	server   *http.Server
	store    *storage.Store
	aiBridge *ai.Bridge // non-nil when AI brain is managed by this server
	aiClient *ai.Client // non-nil when AI is available (managed or external)
}

// Option configures the Server.
type Option func(*Server)

// WithAI configures the server to start and manage a Python AI brain subprocess.
// If port is 0, the default brain port (9711) is used.
func WithAI(port int) Option {
	return func(s *Server) {
		s.aiBridge = ai.NewBridge(port)
		s.aiClient = ai.NewClient(s.aiBridge.Address())
	}
}

// WithAIURL configures the server to connect to an already-running AI brain.
func WithAIURL(url string) Option {
	return func(s *Server) {
		s.aiClient = ai.NewClient(url)
	}
}

// New creates a new API server. Returns an error if storage initialization fails.
func New(addr string, opts ...Option) (*Server, error) {
	store, err := storage.New("")
	if err != nil {
		return nil, fmt.Errorf("initializing storage: %w", err)
	}
	s := &Server{addr: addr, store: store}

	for _, opt := range opts {
		opt(s)
	}

	mux := http.NewServeMux()
	s.registerHandlers(mux)

	// Register web dashboard.
	dash := dashboard.New(store)
	dash.RegisterHandlers(mux)

	s.server = &http.Server{Addr: addr, Handler: mux}
	return s, nil
}

// Start begins listening for connections. If a managed AI bridge is configured,
// it is started before accepting HTTP traffic.
func (s *Server) Start(ctx context.Context) error {
	if s.aiBridge != nil {
		log.Println("[server] starting AI brain subprocess...")
		if err := s.aiBridge.Start(ctx); err != nil {
			log.Printf("[server] AI brain failed to start: %v (AI endpoints will return 503)", err)
			// Don't fail the whole server — AI is optional.
			s.aiBridge = nil
			s.aiClient = nil
		} else {
			log.Println("[server] AI brain is ready")
		}
	}
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server and the AI brain if managed.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.aiBridge != nil {
		log.Println("[server] stopping AI brain...")
		if err := s.aiBridge.Stop(); err != nil {
			log.Printf("[server] AI brain stop error: %v", err)
		}
	}
	return s.server.Shutdown(ctx)
}

// AIEnabled reports whether AI endpoints are available.
func (s *Server) AIEnabled() bool {
	return s.aiClient != nil
}

// Handler returns the server's HTTP handler. This is useful for testing
// with httptest without starting a real listener.
func (s *Server) Handler() http.Handler {
	return s.server.Handler
}
