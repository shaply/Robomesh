package http_server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"roboserver/shared"
	"roboserver/shared/event_bus"
	"roboserver/shared/robot_manager"

	"github.com/go-chi/chi/v5"
)

type HTTPServer_t struct {
	rm     robot_manager.RobotManager
	eb     event_bus.EventBus
	router *chi.Mux
	srv    *http.Server
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
		rm:     rm,
		eb:     eb,
		router: r,
		srv:    srv,
	}

	serverErr := make(chan error, 1)
	go func() {
		s.router.Get("/", s.GETHandleHome) // Root handler

		// Register routes
		s.router.Route("/robot", s.RobotRoutes)

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

func (h *HTTPServer_t) GETHandleHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from RoboHub!")

	fmt.Fprintln(w, "Available robots:")
	for _, robot := range h.rm.GetRobots() {
		fmt.Fprintf(w, robot.String()+"\n")
	}
}
