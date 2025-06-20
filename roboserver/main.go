package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"roboserver/http_server"
	"roboserver/mqtt_server"
	"roboserver/shared"
	"sync"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	log.Println("Server is running on the following IPs:")
	// Print local IPs
	localIPs := shared.GetLocalIPs()
	for _, ip := range localIPs {
		log.Printf("%s\n", ip)
	}

	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		http_server.Start(ctx)
	}()

	// Start MQTT server
	wg.Add(1)
	go func() {
		defer wg.Done()
		mqtt_server.Start(ctx)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs // wait for termination signal
	log.Println("Received termination signal, shutting down...")
	cancel()  // cancel the context to stop the server gracefully
	wg.Wait() // wait for all goroutines to finish
	log.Println("Server has shut down gracefully.")
}
