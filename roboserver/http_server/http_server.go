package http_server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"roboserver/http_server/robot"
	"roboserver/shared"

	"github.com/go-chi/chi/v5"
)

func Start(ctx context.Context) error {
	r := chi.NewRouter()
	srv := &http.Server{
		Addr:    ":8080", // Change to your desired port
		Handler: r,
	}

	go func() {
		r.Get("/", handler) // Root handler

		// Register routes
		r.Route("/robot", robot.RobotRoutes)

		log.Println("Starting HTTP server on", srv.Addr)
		srv.ListenAndServe()
	}()

	<-ctx.Done() // wait for cancellation
	log.Println("Shutting down HTTP server...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Println("Error shutting down HTTP server:", err)
		return fmt.Errorf("error shutting down HTTP server: %w", err)
	}

	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from RoboHub!")

	fmt.Fprintln(w, "Available robots:")
	for id, robot := range shared.Robots {
		fmt.Fprintf(w, "Robot ID: %d, Name: %s, Status: %s\n", id, robot.Name, robot.Status)
	}
}
