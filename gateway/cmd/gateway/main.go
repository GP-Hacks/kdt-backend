package main

import (
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-commons/prettylogger"
	"github.com/GP-Hack/kdt2024-gateway/config"
	grpc_clients "github.com/GP-Hack/kdt2024-gateway/internal/grpc-clients/chat"
	"github.com/GP-Hack/kdt2024-gateway/internal/http-server/handlers/chat"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	cfg := config.MustLoad()
	log := prettylogger.SetupLogger(cfg.Env)
	log.Info("Configuration loaded")
	log.Info("Logger loaded")

	chatClient, err := grpc_clients.SetupChatClient(cfg.ChatAddress)
	if err != nil {
		log.Error("Error setting up ChatClient", slog.String("address", cfg.ChatAddress), slog.String("error", err.Error()))
		os.Exit(1)
	}

	router := setupRouter(log, chatClient)
	startServer(cfg, router, log)
}

func setupRouter(log *slog.Logger, chatClient proto.ChatServiceClient) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Post("/api/chat/ask", chat.NewSendMessageHandler(log, chatClient))
	log.Info("Router successfully created")
	return router
}

func startServer(cfg *config.Config, router *chi.Mux, log *slog.Logger) {
	srv := http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		WriteTimeout: cfg.Timeout,
		ReadTimeout:  cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	log.Info("Starting server", slog.String("address", cfg.Address))
	if err := srv.ListenAndServe(); err != nil {
		log.Error("Error starting server", slog.Any("error", err))
	}
	log.Error("Server shutdown", slog.String("address", cfg.Address))
}
