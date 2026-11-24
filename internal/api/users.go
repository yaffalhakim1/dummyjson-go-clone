package api

import "net/http"

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	opts := parseListOptions(r)
	page := applyList(s.store.Rows.Users, opts)

	s.writeJSON(w, http.StatusOK, map[string]any{
		"users": page,
		"total": len(s.store.Rows.Users),
		"skip":  opts.skip,
		"limit": len(page),
	})
}

func (s *Server) handleFilterUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	key := query.Get("key")
	value := query.Get("value")

	matched := filterRows(s.store.Rows.Users, func(row map[string]any) bool {
		return matchesFilter(row, key, value)
	})

	opts := parseListOptions(r)
	page := applyList(matched, opts)

	s.writeJSON(w, http.StatusOK, map[string]any{
		"users": page,
		"total": len(matched),
		"skip":  opts.skip,
		"limit": len(page),
	})
}

func (s *Server) handleUserPosts(w http.ResponseWriter, r *http.Request) {
	s.writeRelated(w, r, "posts", "userId", "id", s.store.Rows.Posts)
}

// Read the id from the path. Keep only the carts for that user.
func (s *Server) handleUserCartsByID(w http.ResponseWriter, r *http.Request) {
	s.writeRelated(w, r, "carts", "userId", "id", s.store.Rows.Carts)
}

// Read the id from the path. Keep only the todos for that user.
func (s *Server) handleUserTodos(w http.ResponseWriter, r *http.Request) {
	s.writeRelated(w, r, "todos", "userId", "id", s.store.Rows.Todos)
}
