package api

import (
	"dummyjson/internal/auth"
	"net/http"
	"time"
)

type contextKey string

const claimsKey contextKey = "claims"

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type tokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type loginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	Gender       string `json:"gender"`
	Image        string `json:"image"`
}

func userString(row map[string]any, key string) string {
	value, _ := row[key].(string)
	return value
}

func userID(row map[string]any) int {
	value, _ := row["id"].(float64)
	return int(value)
}

func claimsFor(user map[string]any) auth.Claims {
	return auth.Claims{
		ID:        userID(user),
		Username:  userString(user, "username"),
		Email:     userString(user, "email"),
		FirstName: userString(user, "firstName"),
		LastName:  userString(user, "lastName"),
		Gender:    userString(user, "gender"),
		Image:     userString(user, "image"),
	}
}

func (s *Server) signPair(claims auth.Claims) (string, string, error) {
	access, err := s.tokens.Sign(claims, 60*time.Minute)
	if err != nil {
		return "", "", err
	}
	refresh, err := s.tokens.Sign(claims, 7*24*time.Hour)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

// Find the user by name and password. Sign two tokens for them. Send the
// tokens and the user's public fields back.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input loginRequest

	if err := decodeBody(w, r, &input); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid credentials")
		return
	}

	for _, user := range s.store.Rows.Users {
		if userString(user, "username") != input.Username || userString(user, "password") != input.Password {
			continue
		}
		claims := claimsFor(user)
		access, refresh, err := s.signPair(claims)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		s.writeJSON(w, http.StatusOK, loginResponse{
			AccessToken:  access,
			RefreshToken: refresh,
			ID:           claims.ID,
			Username:     claims.Username,
			Email:        claims.Email,
			FirstName:    claims.FirstName,
			LastName:     claims.LastName,
			Gender:       claims.Gender,
			Image:        claims.Image,
		})
		return
	}
	s.writeError(w, http.StatusBadRequest, "Invalid credentials")
}

// Read the claims the auth guard put on the request. Find that user and
// send back the whole row.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "Access Token is required")
		return
	}
	user, err := s.store.FindUser(claims.ID)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "Access Token is required")
		return
	}

	s.writeJSON(w, http.StatusOK, user)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var input refreshRequest
	if err := decodeBody(w, r, &input); err != nil {
		s.writeError(w, http.StatusUnauthorized, "Access Token is required")
		return
	}

	claims, err := s.tokens.Verify(input.RefreshToken)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "Access Token is required")
		return
	}

	access, refresh, err := s.signPair(*claims)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	s.writeJSON(w, http.StatusOK, tokenResponse{AccessToken: access, RefreshToken: refresh})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}
