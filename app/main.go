package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 1. Load the provider configuration from config.yaml
	config, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("FATAL: Failed to load configuration: %v", err)
	}
	log.Println("Configuration loaded.")

	// 2. Create the balancer and initialize it with our providers
	balancer := NewBalancer(config.Providers)
	log.Println("Balancer initialized.")

	// 3. Start the background health checker to monitor providers
	go balancer.StartHealthChecks()
	log.Println("Health checker started.")

	// 4. Create and start the main HTTP server
	server := NewServer(":8080", balancer)
	go func() {
		log.Println("Heimdall Go application starting on port 8080...")
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("FATAL: Server failed: %v", err)
		}
	}()

	// 5. Wait for a shutdown signal to gracefully exit
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
}
