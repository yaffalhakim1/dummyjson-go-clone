package api

import (
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
)

// Send a page of this resource through the shared pipeline. The resource
// name becomes the envelope key.
func (s *Server) listResource(resource string, rows []map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		opts := parseListOptions(r)
		page := applyList(rows, opts)
		s.writeJSON(w, http.StatusOK, map[string]any{
			resource: page,
			"total":  len(rows),
			"skip":   opts.skip,
			"limit":  len(page),
		})
	}
}

// Put the rows into a map by id, so a detail lookup is one step.
func indexRows(rows []map[string]any) map[int]map[string]any {
	index := make(map[int]map[string]any, len(rows))
	for _, row := range rows {
		id, ok := row["id"].(float64)
		if ok {
			index[int(id)] = row
		}
	}
	return index
}

// Find the row whose id is in the path. Send it, or a 404 in upstream wording.
func (s *Server) getResource(resource, display string, rows []map[string]any) http.HandlerFunc {
	byID := indexRows(rows)
	return func(w http.ResponseWriter, r *http.Request) {
		rawID := r.PathValue("id")
		id, err := strconv.Atoi(rawID)
		if err != nil {
			s.writeNotFound(w, display, rawID)
			return
		}
		row, ok := byID[id]
		if !ok {
			s.writeNotFound(w, display, rawID)
			return
		}
		s.writeJSON(w, http.StatusOK, row)
	}
}

// Decide whether one field matches the raw value from the path. A number is
// compared as a number. A list is checked item by item, ignoring case.
func fieldMatches(row map[string]any, field, raw string) bool {
	value, ok := row[field]
	if !ok {
		return false
	}
	switch typed := value.(type) {
	case string:
		return strings.EqualFold(typed, raw)
	case float64:
		number, err := strconv.Atoi(raw)
		return err == nil && int(typed) == number
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.EqualFold(text, raw) {
				return true
			}
		}
	}
	return false
}

// Keep only the rows whose field matches the path value. Then page and send.
// This serves categories, tags, meal types and foreign keys.
func (s *Server) listByField(resource, field, pathParam string, rows []map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := r.PathValue(pathParam)
		matched := filterRows(rows, func(row map[string]any) bool {
			return fieldMatches(row, field, raw)
		})
		opts := parseListOptions(r)
		page := applyList(matched, opts)
		s.writeJSON(w, http.StatusOK, map[string]any{
			resource: page,
			"total":  len(matched),
			"skip":   opts.skip,
			"limit":  len(page),
		})
	}
}

// Pick rows at random. If the path ends in a number, send that many as a
// list. Otherwise send one row.
func (s *Server) writeRandom(w http.ResponseWriter, r *http.Request, rows []map[string]any) {
	if len(rows) == 0 {
		s.writeError(w, http.StatusNotFound, "Not found")
		return
	}
	raw := r.PathValue("n")
	count := 1
	if raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			s.writeError(w, http.StatusNotFound, "Not found")
			return
		}
		count = parsed
	}
	if count > len(rows) {
		count = len(rows)
	}
	order := rand.Perm(len(rows))
	if raw == "" {
		s.writeJSON(w, http.StatusOK, rows[order[0]])
		return
	}
	picked := make([]map[string]any, 0, count)
	for i := 0; i < count; i++ {
		picked = append(picked, rows[order[i]])
	}
	s.writeJSON(w, http.StatusOK, picked)
}

// Serve the three nested post paths. The first wildcard decides which view.
func (s *Server) handlePostsNested(w http.ResponseWriter, r *http.Request) {
	switch r.PathValue("section") {
	case "tag":
		s.listByField("posts", "tags", "value", s.store.Rows.Posts)(w, r)
	case "user":
		s.listByField("posts", "userId", "value", s.store.Rows.Posts)(w, r)
	default:
		s.handlePostComments(w, r)
	}
}

// Say yes if any named field contains the search words, ignoring case.
func matchesAnyField(row map[string]any, query string, fields ...string) bool {
	needle := strings.ToLower(query)
	for _, field := range fields {
		text, ok := row[field].(string)
		if ok && strings.Contains(strings.ToLower(text), needle) {
			return true
		}
	}
	return false
}

// Keep posts whose title or body contains the search words. Then page and send.
func (s *Server) handleSearchPosts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	matched := filterRows(s.store.Rows.Posts, func(row map[string]any) bool {
		return matchesAnyField(row, query, "title", "body")
	})
	opts := parseListOptions(r)
	page := applyList(matched, opts)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"posts": page,
		"total": len(matched),
		"skip":  opts.skip,
		"limit": len(page),
	})
}

// Keep recipes whose name contains the search words. Then page and send.
func (s *Server) handleSearchRecipes(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	matched := filterRows(s.store.Rows.Recipes, func(row map[string]any) bool {
		return matchesAnyField(row, query, "name")
	})
	opts := parseListOptions(r)
	page := applyList(matched, opts)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"recipes": page,
		"total":   len(matched),
		"skip":    opts.skip,
		"limit":   len(page),
	})
}

// Send a derived list that was built once at boot.
func (s *Server) handleProductCategories(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, s.store.Derived.ProductCategories)
}

// Send the product category slugs built at boot.
func (s *Server) handleProductCategoryList(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, s.store.Derived.ProductCategoryList)
}

// Send the post tag objects built at boot.
func (s *Server) handlePostTags(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, s.store.Derived.PostTags)
}

// Send the post tag slugs built at boot.
func (s *Server) handlePostTagList(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, s.store.Derived.PostTagList)
}

// Send the recipe tag strings built at boot.
func (s *Server) handleRecipeTags(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, s.store.Derived.RecipeTags)
}
