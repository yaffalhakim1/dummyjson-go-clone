package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func logRequests(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start).String(),
		)
	})
}

func (s *Server) withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if problem := recover(); problem != nil {
				s.logger.Error("panic recovered", "path", r.URL.Path, "problem", problem)
				s.writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withDelay(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("delay")

		if raw != "" {
			if milliseconds, err := strconv.Atoi(raw); err == nil && milliseconds > 0 {
				duration := time.Duration(milliseconds) * time.Millisecond
				if duration > 10*time.Second {
					duration = 10 * time.Second
				}

				time.Sleep(duration)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			s.writeError(w, http.StatusUnauthorized, "Access Token is required")
			return
		}

		claims, err := s.tokens.Verify(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			s.writeError(w, http.StatusUnauthorized, "Access Token is required")
			return
		}

		withClaims := r.WithContext(context.WithValue(r.Context(), claimsKey, claims))
		next.ServeHTTP(w, withClaims)
	})
}

// Discard the body of the mux's own error handler. Keep its headers so an
// Allow header can be passed through.
type discardWriter struct {
	header http.Header
}

func (writer discardWriter) Header() http.Header {
	return writer.header
}

func (writer discardWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func (writer discardWriter) WriteHeader(int) {}

// When the mux reports no match, it writes plain text. Turn that into the
// same JSON error shape used everywhere else.
func (s *Server) withJSONErrors(mux *http.ServeMux, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, pattern := mux.Handler(r)
		if pattern != "" {
			next.ServeHTTP(w, r)
			return
		}

		header := http.Header{}
		recorder := &statusRecorder{ResponseWriter: discardWriter{header: header}, status: http.StatusOK}
		handler.ServeHTTP(recorder, r)

		if allow := header.Get("Allow"); allow != "" {
			w.Header().Set("Allow", allow)
		}
		s.writeError(w, recorder.status, http.StatusText(recorder.status))
	})
}
