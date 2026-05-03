package app

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"time"

	"payment-service/internal/messaging"
	"payment-service/internal/repository"
	grpctransport "payment-service/internal/transport/grpc"
	"payment-service/internal/usecase"

	paymentv1 "github.com/Temych228/ap2-protos-go/payment/v1"
	"google.golang.org/grpc"

	_ "github.com/lib/pq"
)

type App struct {
	DB        *sql.DB
	Server    *grpc.Server
	Lis       net.Listener
	Publisher messaging.PaymentEventPublisher
}

func New() (*App, error) {
	dbURL := os.Getenv("PSdbURL")
	if dbURL == "" {
		dbURL = os.Getenv("DB_URL")
	}
	if dbURL == "" {
		return nil, fmt.Errorf("DB_URL is not set")
	}

	grpcAddr := os.Getenv("GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = os.Getenv("PAYMENT_GRPC_ADDR")
	}
	if grpcAddr == "" {
		grpcAddr = ":50051"
	}
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
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
	if err := runMigration(db, "migrations/001_create_payments.sql"); err != nil {
		_ = db.Close()
		return nil, err
	}

	publisher, err := messaging.NewRabbitMQPublisher(rabbitURL)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	repo := repository.NewPaymentRepository(db)
	paymentUC := usecase.NewPaymentUsecase(repo, publisher)
	paymentServer := grpctransport.NewPaymentServer(paymentUC)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		_ = publisher.Close()
		_ = db.Close()
		return nil, err
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(grpctransport.LoggingUnaryInterceptor),
	)

	paymentv1.RegisterPaymentServiceServer(server, paymentServer)

	return &App{
		DB:        db,
		Server:    server,
		Lis:       lis,
		Publisher: publisher,
	}, nil
}

func (a *App) Run() error {
	return a.Server.Serve(a.Lis)
}

func (a *App) Close() error {
	if a.Server != nil {
		a.Server.GracefulStop()
	}
	if a.Publisher != nil {
		_ = a.Publisher.Close()
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
