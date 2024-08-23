package main

import (
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
	"log/slog"
	"net/http"
	"os"
)

func main() {
	cfg := config.MustLoad()
	log := prettylogger.SetupLogger(cfg.Env)
	log.Info("Configuration loaded")
	log.Info("Logger loaded")

	err := storage.Connect(cfg.MongoDBPath, cfg.MongoDBName, cfg.MongoDBCollection)
	if err != nil {
		log.Error("Error connecting to MongoDB", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("Connected to MongoDB")

	chatClient, err := chatclient.SetupChatClient(cfg.ChatAddress)
	if err != nil {
		log.Error("Error setting up ChatClient", slog.String("address", cfg.ChatAddress), slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("Connected to ChatService", slog.String("address", cfg.ChatAddress))

	placesClient, err := placesclient.SetupPlacesClient(cfg.PlacesAddress)
	if err != nil {
		log.Error("Error setting up PlacesClient", slog.String("address", cfg.PlacesAddress), slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("Connected to PlacesService", slog.String("address", cfg.PlacesAddress))

	charityClient, err := charityclient.SetupCharityClient(cfg.CharityAddress)
	if err != nil {
		log.Error("Error setting up CharityClient", slog.String("address", cfg.CharityAddress), slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("Connected to CharityService", slog.String("address", cfg.CharityAddress))

	votesClient, err := votesclient.SetupVotesClient(cfg.VotesAddress)
	if err != nil {
		log.Error("Error setting up VotesClient", slog.String("address", cfg.VotesAddress), slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("Connected to VotesService", slog.String("address", cfg.VotesAddress))

	router := setupRouter(log, chatClient, placesClient, charityClient, votesClient)
	startServer(cfg, router, log)
}

func setupRouter(log *slog.Logger, chatClient proto.ChatServiceClient, placesClient proto.PlacesServiceClient, charityClient proto.CharityServiceClient, votesclient proto.VotesServiceClient) *chi.Mux {
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

	router.Get("/api/votes", votes.NewGetVotesHandler(log, votesclient))
	router.Post("/api/votes/get", votes.NewGetVoteInfoHandler(log, votesclient))
	router.Post("/api/votes/rate", votes.NewVoteRateHandler(log, votesclient))
	router.Post("/api/votes/petition", votes.NewVotePetitionHandler(log, votesclient))
	router.Post("/api/votes/choice", votes.NewVoteChoiceHandler(log, votesclient))

	log.Info("Router successfully created")
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
	log.Info("Starting server", slog.String("address", cfg.LocalAddress))
	if err := srv.ListenAndServe(); err != nil {
		log.Error("Error starting server", slog.Any("error", err))
	}
	log.Error("Server shutdown", slog.String("address", cfg.LocalAddress))
}
