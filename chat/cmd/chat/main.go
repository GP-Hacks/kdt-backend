package main

import (
	"github.com/GP-Hack/kdt2024-chat/config"
	"github.com/GP-Hack/kdt2024-chat/internal/grpc-server/handler"
	"github.com/GP-Hack/kdt2024-chat/internal/storage"
	"github.com/GP-Hack/kdt2024-commons/prettylogger"
	"google.golang.org/grpc"
	"log/slog"
	"net"
)

func main() {
	cfg := config.MustLoad()
	log := prettylogger.SetupLogger(cfg.Env)
	log.Info("Configuration loaded")
	log.Info("Logger loaded")

	grpcServer := grpc.NewServer()
	l, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Error("Failed to start listener for ChatService", slog.String("error", err.Error()), slog.String("address", cfg.Address))
		return
	}
	defer func(l net.Listener) {
		_ = l.Close()
	}(l)

	storage := storage.NewRedisStorage(cfg.RedisAddress, 1)
	handler.NewGRPCHandler(grpcServer, storage, log)
	if err := grpcServer.Serve(l); err != nil {
		log.Error("Error serving gRPC server for ChatService", slog.String("address", cfg.Address), slog.String("error", err.Error()))
	}
}
