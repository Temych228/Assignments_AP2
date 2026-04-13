package app

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"time"

	"payment-service/internal/repository"
	grpctransport "payment-service/internal/transport/grpc"
	"payment-service/internal/usecase"

	paymentv1 "github.com/Temych228/ap2-protos-go/payment/v1"
	"google.golang.org/grpc"

	_ "github.com/lib/pq"
)

type App struct {
	DB     *sql.DB
	Server *grpc.Server
	Lis    net.Listener
}

func New() (*App, error) {
	dbURL := os.Getenv("PSdbURL")
	if dbURL == "" {
		return nil, fmt.Errorf("PSdbURL is not set")
	}

	grpcAddr := os.Getenv("PAYMENT_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":50051"
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

	repo := repository.NewPaymentRepository(db)
	paymentUC := usecase.NewPaymentUsecase(repo)
	paymentServer := grpctransport.NewPaymentServer(paymentUC)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(grpctransport.LoggingUnaryInterceptor),
	)

	paymentv1.RegisterPaymentServiceServer(server, paymentServer)

	return &App{
		DB:     db,
		Server: server,
		Lis:    lis,
	}, nil
}

func (a *App) Run() error {
	return a.Server.Serve(a.Lis)
}

func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
