package main

import (
	"flag"
	"github.com/GP-Hack/kdt2024-commons/prettylogger"
	"github.com/GP-Hack/kdt2024-places/config"
	"github.com/GP-Hack/kdt2024-places/internal/grpc-server/handler"
	"github.com/GP-Hack/kdt2024-places/internal/storage"
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
		log.Error("Failed to start listener for PlacesService", slog.String("error", err.Error()), slog.String("address", cfg.Address))
		return
	}
	defer func(l net.Listener) {
		_ = l.Close()
	}(l)

	var path string
	flag.StringVar(&path, "path", "", "postgres://username:password@host:port/dbname")
	flag.Parse()
	if path == "" {
		log.Error("No storage_path provided")
		return
	}
	path = path + "?sslmode=disable"

	storage, err := storage.NewPostgresStorage(path)
	if err != nil {
		log.Error("Failed to connect to Postgres", slog.String("error", err.Error()), slog.String("storage_path", path))
		return
	}
	log.Info("Postgres connected")
	defer storage.Close()

	handler.NewGRPCHandler(grpcServer, storage, log)
	if err := grpcServer.Serve(l); err != nil {
		log.Error("Error serving gRPC server for PlacesService", slog.String("address", cfg.Address), slog.String("error", err.Error()))
		return
	}
}
