package app

import (
	"database/sql"
	"fmt"
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
	Router        *gin.Engine
	DB            *sql.DB
	PaymentClient *clients.PaymentClientGRPC
}

func New() (*App, error) {
	dbURL := os.Getenv("OSdbURL")
	if dbURL == "" {
		return nil, fmt.Errorf("OSdbURL is not set")
	}

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

	paymentAddr := os.Getenv("PAYMENT_GRPC_ADDR")
	if paymentAddr == "" {
		paymentAddr = "localhost:50051"
	}

	paymentClient, err := clients.NewPaymentClientGRPC(paymentAddr)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	orderRepo := repository.NewOrderRepository(db)
	orderUC := usecase.NewOrderUsecase(orderRepo, paymentClient)
	handler := httpadapter.NewHandler(orderUC)

	router := gin.Default()
	router.POST("/orders", handler.CreateOrder)
	router.GET("/orders/stats", handler.GetOrderStats)
	router.GET("/orders/:id", handler.GetOrder)
	router.PATCH("/orders/:id/cancel", handler.CancelOrder)

	return &App{
		Router:        router,
		DB:            db,
		PaymentClient: paymentClient,
	}, nil
}

func (a *App) Close() error {
	if a.PaymentClient != nil {
		_ = a.PaymentClient.Close()
	}
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
