package grpc_clients

import (
	"context"
	"fmt"
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

func SetupPlacesClient(address string) (proto.PlacesServiceClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("Failed to create gRPC connection with places service: %w", err)
	}
	placesClient := proto.NewPlacesServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	healthResponse, err := placesClient.HealthCheck(ctx, &proto.HealthCheckRequest{})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("Health check failed: %w", err)
	}

	if !healthResponse.IsHealthy {
		_ = conn.Close()
		return nil, fmt.Errorf("Places service is not healthy")
	}

	return placesClient, nil
}
