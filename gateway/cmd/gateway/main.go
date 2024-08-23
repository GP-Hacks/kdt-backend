package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-commons/prettylogger"
	"github.com/GP-Hack/kdt2024-gateway/config"
	charityclient "github.com/GP-Hack/kdt2024-gateway/internal/grpc-clients/charity"
	chatclient "github.com/GP-Hack/kdt2024-gateway/internal/grpc-clients/chat"
	placesclient "github.com/GP-Hack/kdt2024-gateway/internal/grpc-clients/places"
	votesclient "github.com/GP-Hack/kdt2024-gateway/internal/grpc-clients/votes"
	"github.com/GP-Hack/kdt2024-gateway/internal/http-server/handlers/charity"
	"github.com/GP-Hack/kdt2024-gateway/internal/http-server/handlers/chat"
	"github.com/GP-Hack/kdt2024-gateway/internal/http-server/handlers/places"
	"github.com/GP-Hack/kdt2024-gateway/internal/http-server/handlers/tokens"
	"github.com/GP-Hack/kdt2024-gateway/internal/http-server/handlers/votes"
	"github.com/GP-Hack/kdt2024-gateway/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.MustLoad()
	log := prettylogger.SetupLogger(cfg.Env)

	log.Info("Configuration loaded", slog.String("environment", cfg.Env))
	log.Info("Logger initialized", slog.String("level", cfg.Env))

	if err := connectToMongoDB(cfg, log); err != nil {
		log.Error("Failed to connect to MongoDB", slog.String("error", err.Error()))
		os.Exit(1)
	}

	chatClient, err := setupChatClient(cfg, log)
	if err != nil {
		log.Error("Failed to setup ChatClient", slog.String("address", cfg.ChatAddress), slog.String("error", err.Error()))
		os.Exit(1)
	}

	placesClient, err := setupPlacesClient(cfg, log)
	if err != nil {
		log.Error("Failed to setup PlacesClient", slog.String("address", cfg.PlacesAddress), slog.String("error", err.Error()))
		os.Exit(1)
	}

	charityClient, err := setupCharityClient(cfg, log)
	if err != nil {
		log.Error("Failed to setup CharityClient", slog.String("address", cfg.CharityAddress), slog.String("error", err.Error()))
		os.Exit(1)
	}

	votesClient, err := setupVotesClient(cfg, log)
	if err != nil {
		log.Error("Failed to setup VotesClient", slog.String("address", cfg.VotesAddress), slog.String("error", err.Error()))
		os.Exit(1)
	}

	router := setupRouter(log, chatClient, placesClient, charityClient, votesClient)
	startServer(cfg, router, log)
}

func connectToMongoDB(cfg *config.Config, log *slog.Logger) error {
	log.Debug("Connecting to MongoDB", slog.String("path", cfg.MongoDBPath), slog.String("name", cfg.MongoDBName), slog.String("collection", cfg.MongoDBCollection))
	err := storage.Connect(cfg.MongoDBPath, cfg.MongoDBName, cfg.MongoDBCollection)
	if err != nil {
		return err
	}
	log.Info("Connected to MongoDB", slog.String("path", cfg.MongoDBPath))
	return nil
}

func setupChatClient(cfg *config.Config, log *slog.Logger) (proto.ChatServiceClient, error) {
	log.Debug("Setting up ChatClient", slog.String("address", cfg.ChatAddress))
	client, err := chatclient.SetupChatClient(cfg.ChatAddress, log)
	if err != nil {
		return nil, err
	}
	log.Info("ChatClient setup successfully", slog.String("address", cfg.ChatAddress))
	return client, nil
}

func setupPlacesClient(cfg *config.Config, log *slog.Logger) (proto.PlacesServiceClient, error) {
	log.Debug("Setting up PlacesClient", slog.String("address", cfg.PlacesAddress))
	client, err := placesclient.SetupPlacesClient(cfg.PlacesAddress, log)
	if err != nil {
		return nil, err
	}
	log.Info("PlacesClient setup successfully", slog.String("address", cfg.PlacesAddress))
	return client, nil
}

func setupCharityClient(cfg *config.Config, log *slog.Logger) (proto.CharityServiceClient, error) {
	log.Debug("Setting up CharityClient", slog.String("address", cfg.CharityAddress))
	client, err := charityclient.SetupCharityClient(cfg.CharityAddress, log)
	if err != nil {
		return nil, err
	}
	log.Info("CharityClient setup successfully", slog.String("address", cfg.CharityAddress))
	return client, nil
}

func setupVotesClient(cfg *config.Config, log *slog.Logger) (proto.VotesServiceClient, error) {
	log.Debug("Setting up VotesClient", slog.String("address", cfg.VotesAddress))
	client, err := votesclient.SetupVotesClient(cfg.VotesAddress, log)
	if err != nil {
		return nil, err
	}
	log.Info("VotesClient setup successfully", slog.String("address", cfg.VotesAddress))
	return client, nil
}

func setupRouter(log *slog.Logger, chatClient proto.ChatServiceClient, placesClient proto.PlacesServiceClient, charityClient proto.CharityServiceClient, votesClient proto.VotesServiceClient) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/api/chat/ask", chat.NewSendMessageHandler(log, chatClient))
	router.Post("/api/places/get", places.NewGetPlacesHandler(log, placesClient))
	router.Post("/api/places/buy", places.NewBuyTicketHandler(log, placesClient))
	router.Get("/api/places/categories", places.NewGetCategoriesHandler(log, placesClient))

	router.Post("/api/charity/get", charity.NewGetCollectionsHandler(log, charityClient))
	router.Post("/api/charity/donate", charity.NewDonateHandler(log, charityClient))
	router.Get("/api/charity/categories", charity.NewGetCategoriesHandler(log, charityClient))

	router.Post("/api/user/token", tokens.NewAddTokenHandler(log))

	router.Get("/api/votes", votes.NewGetVotesHandler(log, votesClient))
	router.Post("/api/votes/get", votes.NewGetVoteInfoHandler(log, votesClient))
	router.Post("/api/votes/rate", votes.NewVoteRateHandler(log, votesClient))
	router.Post("/api/votes/petition", votes.NewVotePetitionHandler(log, votesClient))
	router.Post("/api/votes/choice", votes.NewVoteChoiceHandler(log, votesClient))

	log.Info("Router successfully created with defined routes")
	return router
}

func startServer(cfg *config.Config, router *chi.Mux, log *slog.Logger) {
	srv := http.Server{
		Addr:         cfg.LocalAddress,
		Handler:      router,
		WriteTimeout: cfg.Timeout,
		ReadTimeout:  cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	log.Info("Starting HTTP server", slog.String("address", cfg.LocalAddress))
	if err := srv.ListenAndServe(); err != nil {
		log.Error("Server encountered an error", slog.String("address", cfg.LocalAddress), slog.Any("error", err))
		return
	}

	log.Info("Server shutdown gracefully", slog.String("address", cfg.LocalAddress))
}
