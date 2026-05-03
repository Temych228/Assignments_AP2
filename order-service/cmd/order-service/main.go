package main

import (
	"log"
	"net/http"

	"order-service/internal/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}
	defer a.Close()

	go func() {
		log.Println("Order gRPC server running")
		if err := a.RunGRPC(); err != nil {
			log.Fatalf("failed to run grpc server: %v", err)
		}
	}()

	log.Println("Order HTTP server running on :8080")
	if err := a.Router.Run(":8080"); err != nil {
		if err != http.ErrServerClosed {
			log.Fatalf("failed to run http server: %v", err)
		}
	}
}
