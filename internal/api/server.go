package api

import (
	"log/slog"
	"net/http"

	"dummyjson/internal/auth"
	"dummyjson/internal/store"
)

type Server struct {
	store  *store.Store
	logger *slog.Logger
	tokens *auth.Manager
}

func NewServer(st *store.Store, logger *slog.Logger, tokens *auth.Manager) *Server {
	return &Server{store: st, logger: logger, tokens: tokens}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)

	var handler http.Handler = mux
	handler = s.withDelay(handler)
	handler = s.withJSONErrors(mux, handler)
	handler = s.withCORS(handler)
	handler = s.withRecover(handler)
	handler = logRequests(s.logger, handler)

	return handler
}
