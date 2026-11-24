package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func (s *Server) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		s.logger.Error("encode response", "err", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"message": message})
}

func (s *Server) writeNotFound(w http.ResponseWriter, resource, rawID string) {
	s.writeError(w, http.StatusNotFound,
		fmt.Sprintf("%s with id '%s' not found", resource, rawID))
}

// Read the id from the path. Keep only the rows that point at that id.
// Then apply the page options and send the envelope under the given key.
func (s *Server) writeRelated(w http.ResponseWriter, r *http.Request, resource, foreignKey, pathParam string, rows []map[string]any) {
	rawID := r.PathValue(pathParam)
	id, err := strconv.Atoi(rawID)
	if err != nil {
		s.writeNotFound(w, resource, rawID)
		return
	}

	matched := filterByID(rows, foreignKey, id)
	opts := parseListOptions(r)
	page := applyList(matched, opts)

	s.writeJSON(w, http.StatusOK, map[string]any{
		resource: page,
		"total":  len(matched),
		"skip":   opts.skip,
		"limit":  len(page),
	})
}

const maxBodyBytes = 300 << 10

func decodeBody(w http.ResponseWriter, r *http.Request, target any) error {
	body := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer body.Close()
	return json.NewDecoder(body).Decode(target)
}
