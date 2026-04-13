package grpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func LoggingUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	md, _ := metadata.FromIncomingContext(ctx)
	log.Printf("gRPC request started: method=%s metadata=%v", info.FullMethod, md)

	resp, err := handler(ctx, req)

	log.Printf(
		"gRPC request finished: method=%s duration=%s error=%v",
		info.FullMethod,
		time.Since(start),
		err,
	)

	return resp, err
}
