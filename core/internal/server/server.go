package server

import (
	"context"
	"net/http"

	"github.com/mkaganm/probex/internal/dashboard"
	"github.com/mkaganm/probex/internal/storage"
)

// Server provides a REST API for SDK clients (Java, JS/TS).
type Server struct {
	addr   string
	server *http.Server
	store  *storage.Store
}

// New creates a new API server.
func New(addr string) *Server {
	store, _ := storage.New("")
	s := &Server{addr: addr, store: store}
	mux := http.NewServeMux()
	s.registerHandlers(mux)

	// Register web dashboard.
	dash := dashboard.New(store)
	dash.RegisterHandlers(mux)

	s.server = &http.Server{Addr: addr, Handler: mux}
	return s
}

// Start begins listening for connections.
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
