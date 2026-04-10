package app

import (
	"database/sql"
	"os"
	"time"

	"order-service/internal/clients"
	"order-service/internal/repository"
	httpadapter "order-service/internal/transport/http"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type App struct {
	Router *gin.Engine
	DB     *sql.DB
}

func New() (*App, error) {
	dbURL := os.Getenv("OSdbURL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	orderRepo := repository.NewOrderRepository(db)

	paymentURL := os.Getenv("PAYMENT_SERVICE_URL")
	if paymentURL == "" {
		paymentURL = "http://localhost:8081"
	}

	paymentClient := clients.NewPaymentClientHTTP(paymentURL)
	orderUC := usecase.NewOrderUsecase(orderRepo, paymentClient)
	handler := httpadapter.NewHandler(orderUC)

	router := gin.Default()
	router.POST("/orders", handler.CreateOrder)
	router.GET("/orders/stats", handler.GetOrderStats)
	router.GET("/orders/:id", handler.GetOrder)
	router.PATCH("/orders/:id/cancel", handler.CancelOrder)

	return &App{
		Router: router,
		DB:     db,
	}, nil
}
