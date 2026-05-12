package app

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"order-service/internal/cache"
	"order-service/internal/clients"
	"order-service/internal/middleware"
	"order-service/internal/repository"
	grpctransport "order-service/internal/transport/grpc"
	httpadapter "order-service/internal/transport/http"
	"order-service/internal/usecase"

	orderv1 "github.com/Temych228/ap2-protos-go/order/v1"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type App struct {
	Router        *gin.Engine
	DB            *sql.DB
	Redis         *redis.Client
	PaymentClient *clients.PaymentClientGRPC
	GRPCServer    *grpc.Server
	GRPCListener  net.Listener
}

func New() (*App, error) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DB_URL is not set")
	}

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	cacheTTL := mustDuration(os.Getenv("ORDER_CACHE_TTL"), 5*time.Minute)
	rateLimit := mustInt(os.Getenv("RATE_LIMIT_PER_MINUTE"), 10)

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

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		_ = db.Close()
		_ = rdb.Close()
		return nil, err
	}

	paymentAddr := os.Getenv("PAYMENT_GRPC_ADDR")
	if paymentAddr == "" {
		paymentAddr = "localhost:50051"
	}

	paymentClient, err := clients.NewPaymentClientGRPC(paymentAddr)
	if err != nil {
		_ = rdb.Close()
		_ = db.Close()
		return nil, err
	}

	orderRepo := repository.NewOrderRepository(db, dbURL)
	orderCache := cache.NewRedisOrderCache(rdb)
	orderUC := usecase.NewOrderUsecase(orderRepo, paymentClient, orderCache, cacheTTL)
	handler := httpadapter.NewHandler(orderUC)
	orderServer := grpctransport.NewOrderServer(orderUC)

	router := gin.Default()
	router.Use(middleware.NewRateLimiter(rdb, rateLimit, time.Minute).Middleware())

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
		_ = rdb.Close()
		_ = db.Close()
		return nil, err
	}

	grpcServer := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(grpcServer, orderServer)

	return &App{
		Router:        router,
		DB:            db,
		Redis:         rdb,
		PaymentClient: paymentClient,
		GRPCServer:    grpcServer,
		GRPCListener:  lis,
	}, nil
}

func (a *App) RunGRPC() error {
	return a.GRPCServer.Serve(a.GRPCListener)
}

func (a *App) Close() error {
	if a.GRPCServer != nil {
		a.GRPCServer.GracefulStop()
	}
	if a.PaymentClient != nil {
		_ = a.PaymentClient.Close()
	}
	if a.Redis != nil {
		_ = a.Redis.Close()
	}
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}

func runMigration(db *sql.DB, path string) error {
	query, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = db.Exec(string(query))
	return err
}

func mustDuration(v string, def time.Duration) time.Duration {
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func mustInt(v string, def int) int {
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
