package main

import (
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-commons/prettylogger"
	"github.com/GP-Hack/kdt2024-gateway/config"
	chat_client "github.com/GP-Hack/kdt2024-gateway/internal/grpc-clients/chat"
	places_client "github.com/GP-Hack/kdt2024-gateway/internal/grpc-clients/places"
	"github.com/GP-Hack/kdt2024-gateway/internal/http-server/handlers/chat"
	"github.com/GP-Hack/kdt2024-gateway/internal/http-server/handlers/places"
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

	chatClient, err := chat_client.SetupChatClient(cfg.ChatAddress)
	if err != nil {
		log.Error("Error setting up ChatClient", slog.String("address", cfg.ChatAddress), slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("Connected to ChatService", slog.String("address", cfg.ChatAddress))

	placesClient, err := places_client.SetupPlacesClient(cfg.PlacesAddress)
	if err != nil {
		log.Error("Error setting up PlacesClient", slog.String("address", cfg.PlacesAddress), slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("Connected to PlacesService", slog.String("address", cfg.PlacesAddress))

	router := setupRouter(log, chatClient, placesClient)
	startServer(cfg, router, log)
}

func setupRouter(log *slog.Logger, chatClient proto.ChatServiceClient, placesClient proto.PlacesServiceClient) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Post("/api/chat/ask", chat.NewSendMessageHandler(log, chatClient))
	router.Post("/api/places/get", places.NewGetPlacesHandler(log, placesClient))
	router.Post("/api/places/buy", places.NewBuyTicketHandler(log, placesClient))
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
