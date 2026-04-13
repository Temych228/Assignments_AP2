package main

import (
	"log"

	"payment-service/internal/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}
	defer a.Close()

	log.Println("Payment Service running")
	if err := a.Run(); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
