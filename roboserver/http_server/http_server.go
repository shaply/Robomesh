package http_server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"roboserver/comms"
	"roboserver/database"
	"roboserver/http_server/http_events"
	"roboserver/http_server/http_websocket"
	"roboserver/shared"
	"time"

	"github.com/go-chi/chi/v5"
)

type HTTPServer_t struct {
	ctx        context.Context // server-level context for long-lived operations
	bus        comms.Bus
	db         database.DBManager
	router     *chi.Mux
	srv        *http.Server
	sseManager *http_events.EventsManager_t
	wsManager  *http_websocket.Manager
}

func Start(ctx context.Context, bus comms.Bus, db database.DBManager) error {
	r := chi.NewRouter()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", shared.AppConfig.Server.HTTPPort),
		Handler: r,
	}

	s := &HTTPServer_t{
		ctx:        ctx,
		bus:        bus,
		db:         db,
		router:     r,
		srv:        srv,
		sseManager: http_events.NewEventsManager(bus),
		wsManager:  http_websocket.NewManager(bus),
	}

	serverErr := make(chan error, 1)
	go func() {
		// Global middleware
		s.router.Use(s.LoggingMiddleware)
		s.router.Use(s.CORSMiddleware)
		s.router.Use(s.BodySizeLimitMiddleware)

		// Public routes
		s.router.Route("/auth", s.AuthRoutes)
		s.router.Route("/heartbeat", s.HeartbeatRoutes)
		s.router.Route("/plugins", s.PluginRoutes)

		// Semi-public: SSE GET accepts tickets (handles its own auth)
		s.router.Get("/events", s.eventsHandler)
		s.router.Get("/handler/{uuid}/logs", s.streamHandlerLogs) // ticket-based auth

		// Protected routes
		s.router.Group(func(r chi.Router) {
			r.Use(s.SessionValidationMiddleware)
			r.Route("/robot", s.RobotRoutes)
			r.Post("/events/subscribe", s.eventsSubscribeHandler)
			r.Post("/events/unsubscribe", s.eventsUnsubscribeHandler)
			r.Route("/provision", s.ProvisionRoutes)
			r.Route("/ephemeral", s.EphemeralRoutes)
			r.Route("/register", s.RegisterRoutes)
			r.Route("/handler", s.HandlerRoutes)
			r.Get("/ws", s.wsHandler)
		})

		if shared.AppConfig.Server.TLS.Enabled {
			cert, tlsErr := tls.LoadX509KeyPair(
				shared.AppConfig.Server.TLS.CertFile,
				shared.AppConfig.Server.TLS.KeyFile,
			)
			if tlsErr != nil {
				serverErr <- fmt.Errorf("failed to load TLS certificate: %w", tlsErr)
				return
			}
			s.srv.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			}
			shared.DebugPrint("Starting HTTPS server on %s", s.srv.Addr)
			if err := s.srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				serverErr <- fmt.Errorf("error starting HTTPS server: %w", err)
			}
		} else {
			shared.DebugPrint("Starting HTTP server on %s", s.srv.Addr)
			if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				serverErr <- fmt.Errorf("error starting HTTP server: %w", err)
			}
		}
	}()

	select {
	case err := <-serverErr:
		shared.DebugPanic("%v", err)
	case <-ctx.Done():
		shared.DebugPrint("Shutting down HTTP server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer shutdownCancel()
		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			shared.DebugPrint("Error shutting down HTTP server:", err)
			return fmt.Errorf("error shutting down HTTP server: %w", err)
		}
	}

	return nil
}

// wsHandler upgrades to WebSocket for bidirectional communication (event streaming, commands).
func (s *HTTPServer_t) wsHandler(w http.ResponseWriter, r *http.Request) {
	s.wsManager.HandleConnection(w, r)
}

// SessionValidationMiddleware validates session for protected routes.
// Checks both JWT validity and Redis session existence (prevents use after logout).
func (s *HTTPServer_t) SessionValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := s.validateSessionFull(r)
		if session == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs all requests
func (s *HTTPServer_t) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shared.DebugPrint("%s %s from %s", r.Method, r.URL.Path, shared.RedactIP(r.RemoteAddr))
		next.ServeHTTP(w, r)
	})
}

// BodySizeLimitMiddleware caps request bodies to prevent memory exhaustion
// from oversized payloads. Applied globally; individual handlers can set
// tighter limits as needed.
func (s *HTTPServer_t) BodySizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware handles Cross-Origin Resource Sharing with origin whitelist.
func (s *HTTPServer_t) CORSMiddleware(next http.Handler) http.Handler {
	allowed := make(map[string]bool, len(shared.AppConfig.Server.AllowedOrigins))
	for _, o := range shared.AppConfig.Server.AllowedOrigins {
		allowed[o] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
