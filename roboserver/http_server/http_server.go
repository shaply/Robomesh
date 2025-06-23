package http_server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"roboserver/shared/robot_manager"

	"github.com/go-chi/chi/v5"
)

type HTTPServer struct {
	robotHandler *robot_manager.RobotHandler
	router       *chi.Mux
	srv          *http.Server
}

func Start(ctx context.Context, robotHandler *robot_manager.RobotHandler) error {
	r := chi.NewRouter()

	// Get port
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		log.Fatal("HTTP_PORT environment variable is not set")
	}
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}
	defer srv.Shutdown(ctx)

	s := &HTTPServer{
		robotHandler: robotHandler,
		router:       r,
		srv:          srv,
	}

	serverErr := make(chan error, 1)
	go func() {
		s.router.Get("/", s.GETHandleHome) // Root handler

		// Register routes
		s.router.Route("/robot", s.RobotRoutes)

		log.Println("Starting HTTP server on", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil {
			serverErr <- fmt.Errorf("error starting HTTP server: %w", err)
		}
	}()

	select {
	case err := <-serverErr:
		log.Fatal(err)
	case <-ctx.Done():
		log.Println("Shutting down HTTP server...")
		if err := s.srv.Shutdown(ctx); err != nil {
			log.Println("Error shutting down HTTP server:", err)
			return fmt.Errorf("error shutting down HTTP server: %w", err)
		}
	}

	return nil
}

func (h *HTTPServer) GETHandleHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from RoboHub!")

	fmt.Fprintln(w, "Available robots:")
	for _, robot := range h.robotHandler.GetRobots() {
		fmt.Fprintf(w, "Robot ID: %d, Name: %s, Status: %s\n", robot.ID, robot.Name, robot.Status)
	}
}
