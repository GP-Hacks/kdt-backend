package grpc_clients

import (
	"fmt"
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func SetupChatClient(address string) (proto.ChatServiceClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("Failed to create gRPC connection with chat service: %w", err)
	}
	return proto.NewChatServiceClient(conn), nil
}
