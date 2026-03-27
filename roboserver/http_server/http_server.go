package http_server

import (
	"context"
	"fmt"
	"net/http"
	"roboserver/comms"
	"roboserver/database"
	"roboserver/http_server/http_events"
	"roboserver/shared"

	"github.com/go-chi/chi/v5"
)

type HTTPServer_t struct {
	bus        comms.Bus
	db         database.DBManager
	router     *chi.Mux
	srv        *http.Server
	sseManager *http_events.EventsManager_t
}

func Start(ctx context.Context, bus comms.Bus, db database.DBManager) error {
	r := chi.NewRouter()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", shared.AppConfig.Server.HTTPPort),
		Handler: r,
	}
	defer srv.Shutdown(ctx)

	s := &HTTPServer_t{
		bus:        bus,
		db:         db,
		router:     r,
		srv:        srv,
		sseManager: http_events.NewEventsManager(bus),
	}

	serverErr := make(chan error, 1)
	go func() {
		// Global middleware
		s.router.Use(s.LoggingMiddleware)
		s.router.Use(s.CORSMiddleware)

		// Public routes
		s.router.Route("/auth", s.AuthRoutes)

		// Protected routes
		s.router.Group(func(r chi.Router) {
			r.Use(s.SessionValidationMiddleware)
			r.Route("/robot", s.RobotRoutes)
			r.Route("/events", s.EventRoutes)
			r.Route("/provision", s.ProvisionRoutes)
			r.Route("/ephemeral", s.EphemeralRoutes)
			r.Route("/register", s.RegisterRoutes)
		})

		shared.DebugPrint("Starting HTTP server on %s", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil {
			serverErr <- fmt.Errorf("error starting HTTP server: %w", err)
		}
	}()

	select {
	case err := <-serverErr:
		shared.DebugPanic("%v", err)
	case <-ctx.Done():
		shared.DebugPrint("Shutting down HTTP server...")
		if err := s.srv.Shutdown(ctx); err != nil {
			shared.DebugPrint("Error shutting down HTTP server:", err)
			return fmt.Errorf("error shutting down HTTP server: %w", err)
		}
	}

	return nil
}

// SessionValidationMiddleware validates session for protected routes
func (s *HTTPServer_t) SessionValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := GetSessionFromRequest(r)
		if session == nil {
			http.Error(w, "Unauthorized: No session found", http.StatusUnauthorized)
			return
		}

		if err := ValidateSession(session); err != nil {
			http.Error(w, "Unauthorized: Invalid session", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs all requests
func (s *HTTPServer_t) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shared.DebugPrint("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func (s *HTTPServer_t) CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
