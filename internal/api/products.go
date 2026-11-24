package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"dummyjson/internal/store"
)

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	opts := parseListOptions(r)
	page := applyList(s.store.Rows.Products, opts)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"products": page,
		"total":    len(s.store.Rows.Products),
		"skip":     opts.skip,
		"limit":    len(page),
	})
}

func (s *Server) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")

	id, err := strconv.Atoi(rawID)
	if err != nil {
		s.writeNotFound(w, "Product", rawID)
		return
	}

	product, err := s.store.FindProduct(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.writeNotFound(w, "Product", rawID)
			return
		}

		s.writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.writeJSON(w, http.StatusOK, product)
}

func (s *Server) handleSearchProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	matched := filterRows(s.store.Rows.Products, func(row map[string]any) bool {
		return matchesSearch(row, query)
	})

	opts := parseListOptions(r)
	page := applyList(matched, opts)

	s.writeJSON(w, http.StatusOK, map[string]any{
		"products": page,
		"total":    len(matched),
		"skip":     opts.skip,
		"limit":    len(page),
	})
}

func (s *Server) handleAddProduct(w http.ResponseWriter, r *http.Request) {
	input := make(map[string]any)
	if err := decodeBody(w, r, &input); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	product := make(map[string]any, len(input)+1)
	for key, value := range input {
		product[key] = value
	}

	product["id"] = float64(len(s.store.Rows.Products) + 1)

	s.writeJSON(w, http.StatusCreated, product)
}

func (s *Server) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")

	id, err := strconv.Atoi(rawID)
	if err != nil {
		s.writeNotFound(w, "Product", rawID)
		return
	}

	existing, err := s.store.FindProduct(id)

	if err != nil {
		s.writeNotFound(w, "Product", rawID)
		return
	}

	patch := make(map[string]any)
	if err := decodeBody(w, r, &patch); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	merged := make(map[string]any, len(existing)+len(patch))

	for key, value := range existing {
		merged[key] = value
	}

	for key, value := range patch {
		merged[key] = value
	}

	s.writeJSON(w, http.StatusOK, merged)
}

func (s *Server) handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := strconv.Atoi(rawID)

	if err != nil {
		s.writeNotFound(w, "Product", rawID)
		return
	}

	product, err := s.store.FindProduct(id)
	if err != nil {
		s.writeNotFound(w, "Product", rawID)
		return
	}

	result := make(map[string]any, len(product)+2)
	for key, value := range product {
		result[key] = value
	}
	result["isDeleted"] = true
	result["deletedOn"] = time.Now().UTC().Format(time.RFC3339)

	s.writeJSON(w, http.StatusOK, result)
}
