package main

import (
	"log"

	"order-service/internal/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}
	defer a.Close()

	log.Println("Order Service running on :8080")
	if err := a.Router.Run(":8080"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
