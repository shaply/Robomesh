package http_server

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

func Start(ctx context.Context) error {
	srv := &http.Server{Addr: ":8080"}
	http.HandleFunc("/", handler)

	go func() {
		log.Println("HTTP server running on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-ctx.Done() // wait for cancellation
	log.Println("Shutting down HTTP server...")
	srv.Shutdown(context.Background())

	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from RoboHub!")
}
