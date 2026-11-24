package api

import "net/http"

// Read the userId from the path. Keep only the carts for that user.
func (s *Server) handleUserCarts(w http.ResponseWriter, r *http.Request) {
	s.writeRelated(w, r, "carts", "userId", "userId", s.store.Rows.Carts)
}
