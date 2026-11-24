package api

import "net/http"

func (s *Server) handlePostComments(w http.ResponseWriter, r *http.Request) {
	s.writeRelated(w, r, "comments", "postId", "section", s.store.Rows.Comments)
}
