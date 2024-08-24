package main

import (
	"context"
	"net"

	"github.com/GP-Hacks/kdt2024-chat/config"
	"github.com/GP-Hacks/kdt2024-chat/internal/grpc-server/handler"
	"github.com/GP-Hacks/kdt2024-chat/internal/storage"
	"github.com/GP-Hacks/kdt2024-commons/prettylogger"
	"google.golang.org/grpc"
	"log/slog"
)

func main() {
	ctx := context.Background()

	cfg := config.MustLoad()
	log := prettylogger.SetupLogger(cfg.Env)
	log.Info("Configuration and logger initialized", slog.String("environment", cfg.Env))

	redisStorage, err := initRedisStorage(ctx, cfg, log)
	if err != nil {
		log.Error("Failed to initialize Redis storage", slog.String("error", err.Error()))
		return
	}

	if err := startGRPCServer(ctx, cfg, redisStorage, log); err != nil {
		log.Error("gRPC server encountered an error", slog.String("error", err.Error()))
	}
}

func initRedisStorage(ctx context.Context, cfg *config.Config, log *slog.Logger) (*storage.RedisStorage, error) {
	log.Info("Connecting to Redis", slog.String("address", cfg.RedisAddress), slog.Int("db", 1))

	redisStorage, err := storage.NewRedisStorage(cfg.RedisAddress, 1)
	if err != nil {
		log.Error("Failed to connect to Redis", slog.String("address", cfg.RedisAddress), slog.String("error", err.Error()))
		return nil, err
	}

	log.Info("Successfully connected to Redis", slog.String("address", cfg.RedisAddress))
	return redisStorage, nil
}

func startGRPCServer(ctx context.Context, cfg *config.Config, redisStorage *storage.RedisStorage, log *slog.Logger) error {
	log.Info("Starting gRPC server", slog.String("address", cfg.Address))

	grpcServer := grpc.NewServer()
	handler.NewGRPCHandler(grpcServer, redisStorage, log)

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Error("Failed to start TCP listener", slog.String("address", cfg.Address), slog.String("error", err.Error()))
		return err
	}
	log.Info("TCP listener started", slog.String("address", cfg.Address))
	defer func() {
		if err := listener.Close(); err != nil {
			log.Warn("Failed to close TCP listener", slog.String("error", err.Error()))
		} else {
			log.Info("TCP listener closed gracefully")
		}
	}()

	errCh := make(chan error)
	go func() {
		if serveErr := grpcServer.Serve(listener); serveErr != nil {
			errCh <- serveErr
		}
	}()

	log.Info("gRPC server is running", slog.String("address", cfg.Address))

	select {
	case <-ctx.Done():
		log.Info("Received shutdown signal, stopping gRPC server")
		grpcServer.GracefulStop()
	case serveErr := <-errCh:
		log.Error("gRPC server stopped with error", slog.String("error", serveErr.Error()))
		return serveErr
	}

	log.Info("gRPC server stopped gracefully")
	return nil
}
