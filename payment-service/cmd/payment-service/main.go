package main

import (
	"database/sql"
	"log"

	"payment-service/internal/repository"
	"payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("PSdbURL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	repo := repository.NewPaymentRepository(db)
	usecase := usecase.NewPaymentUsecase(repo)
	handler := http.NewHandler(usecase)

	r := gin.Default()

	r.POST("/payments", handler.CreatePayment)
	r.GET("/payments/:order_id", handler.GetPayment)

	log.Println("Payment Service running on :8081")
	r.Run(":8081")
}
