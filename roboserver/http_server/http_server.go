package http_server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"roboserver/http_server/http_events"
	"roboserver/shared"
	"roboserver/shared/event_bus"
	"roboserver/shared/robot_manager"

	"github.com/go-chi/chi/v5"
)

type HTTPServer_t struct {
	rm         robot_manager.RobotManager
	eb         event_bus.EventBus
	router     *chi.Mux
	srv        *http.Server
	sseManager *http_events.EventsManager_t // Server-Sent Events manager for handling SSE connections
}

func Start(ctx context.Context, rm robot_manager.RobotManager, eb event_bus.EventBus) error {
	r := chi.NewRouter()

	// Get port
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		shared.DebugPanic("HTTP_PORT environment variable is not set")
	}
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}
	defer srv.Shutdown(ctx)

	s := &HTTPServer_t{
		rm:         rm,
		eb:         eb,
		router:     r,
		srv:        srv,
		sseManager: http_events.NewEventsManager(eb), // Initialize Server-Sent Events manager
	}

	serverErr := make(chan error, 1)
	go func() {
		// Global middleware (applies to all routes)
		s.router.Use(s.LoggingMiddleware) // Log all requests
		s.router.Use(s.CORSMiddleware)    // Handle CORS for cross-origin requests

		// Public routes (no authentication required)
		s.router.Route("/auth", s.AuthRoutes)

		// Protected routes (require authentication)
		s.router.Group(func(r chi.Router) {
			r.Use(s.SessionValidationMiddleware) // Apply session validation to this group
			r.Route("/robot", s.RobotRoutes)
			r.Route("/events", s.EventRoutes)
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
		// Get session from request (cookie, header, etc.)
		session := GetSessionFromRequest(r)
		if session == nil {
			http.Error(w, "Unauthorized: No session found", http.StatusUnauthorized)
			return
		}

		// Validate the session
		if err := ValidateSession(session); err != nil {
			http.Error(w, "Unauthorized: Invalid session", http.StatusUnauthorized)
			return
		}

		// Session is valid, continue to next handler
		next.ServeHTTP(w, r)
	})
}

// Optional: Logging middleware
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

		// When credentials are included, we must specify exact origins, never "*"
		if origin != "" {
			// Allow the specific requesting origin (for development)
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			// If no Origin header, assume same-origin request from frontend
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400") // Cache preflight for 24 hours

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}
