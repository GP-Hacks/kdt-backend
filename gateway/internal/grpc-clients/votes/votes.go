package grpc_clients

import (
	"context"
	"fmt"
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

func SetupVotesClient(address string) (proto.VotesServiceClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("Failed to create gRPC connection with votes service: %w", err)
	}
	votesClient := proto.NewVotesServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	healthResponse, err := votesClient.HealthCheck(ctx, &proto.HealthCheckRequest{})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("Health check failed: %w", err)
	}

	if !healthResponse.IsHealthy {
		_ = conn.Close()
		return nil, fmt.Errorf("Votes service is not healthy")
	}

	return votesClient, nil
}
