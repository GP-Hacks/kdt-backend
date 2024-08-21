package grpc_clients

import (
	"context"
	"fmt"
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

func SetupCharityClient(address string) (proto.CharityServiceClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("Failed to create gRPC connection with charity service: %w", err)
	}
	charityClient := proto.NewCharityServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	healthResponse, err := charityClient.HealthCheck(ctx, &proto.HealthCheckRequest{})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("Health check failed: %w", err)
	}

	if !healthResponse.IsHealthy {
		_ = conn.Close()
		return nil, fmt.Errorf("Charity service is not healthy")
	}

	return charityClient, nil
}
