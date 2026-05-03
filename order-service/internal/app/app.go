package app

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"time"

	orderv1 "github.com/Temych228/ap2-protos-go/order/v1"
	"order-service/internal/clients"
	"order-service/internal/repository"
	grpctransport "order-service/internal/transport/grpc"
	httpadapter "order-service/internal/transport/http"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

type App struct {
	Router        *gin.Engine
	DB            *sql.DB
	PaymentClient *clients.PaymentClientGRPC
	GRPCServer    *grpc.Server
	GRPCListener  net.Listener
}

func New() (*App, error) {
	dbURL := os.Getenv("OSdbURL")
	if dbURL == "" {
		dbURL = os.Getenv("DB_URL")
	}
	if dbURL == "" {
		return nil, fmt.Errorf("DB_URL is not set")
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
	if err := runMigration(db, "migrations/001_create_orders.sql"); err != nil {
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

	orderRepo := repository.NewOrderRepository(db, dbURL)
	orderUC := usecase.NewOrderUsecase(orderRepo, paymentClient)
	handler := httpadapter.NewHandler(orderUC)
	orderServer := grpctransport.NewOrderServer(orderUC)

	router := gin.Default()
	router.POST("/orders", handler.CreateOrder)
	router.GET("/orders/stats", handler.GetOrderStats)
	router.GET("/orders/:id", handler.GetOrder)
	router.PATCH("/orders/:id/cancel", handler.CancelOrder)

	grpcAddr := os.Getenv("GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":9090"
	}

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		_ = paymentClient.Close()
		_ = db.Close()
		return nil, err
	}

	grpcServer := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(grpcServer, orderServer)

	return &App{
		Router:        router,
		DB:            db,
		PaymentClient: paymentClient,
		GRPCServer:    grpcServer,
		GRPCListener:  lis,
	}, nil
}

func (a *App) RunGRPC() error {
	return a.GRPCServer.Serve(a.GRPCListener)
}

func runMigration(db *sql.DB, path string) error {
	query, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = db.Exec(string(query))
	return err
}

func (a *App) Close() error {
	if a.GRPCServer != nil {
		a.GRPCServer.GracefulStop()
	}
	if a.PaymentClient != nil {
		_ = a.PaymentClient.Close()
	}
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
