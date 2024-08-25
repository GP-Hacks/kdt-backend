package main

import (
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
	"github.com/GP-Hacks/kdt2024-commons/prettylogger"
	"github.com/GP-Hacks/kdt2024-gateway/config"
	charityclient "github.com/GP-Hacks/kdt2024-gateway/internal/grpc-clients/charity"
	chatclient "github.com/GP-Hacks/kdt2024-gateway/internal/grpc-clients/chat"
	placesclient "github.com/GP-Hacks/kdt2024-gateway/internal/grpc-clients/places"
	votesclient "github.com/GP-Hacks/kdt2024-gateway/internal/grpc-clients/votes"
	"github.com/GP-Hacks/kdt2024-gateway/internal/http-server/handlers/charity"
	"github.com/GP-Hacks/kdt2024-gateway/internal/http-server/handlers/chat"
	"github.com/GP-Hacks/kdt2024-gateway/internal/http-server/handlers/places"
	"github.com/GP-Hacks/kdt2024-gateway/internal/http-server/handlers/tokens"
	"github.com/GP-Hacks/kdt2024-gateway/internal/http-server/handlers/votes"
	"github.com/GP-Hacks/kdt2024-gateway/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	httpSwagger "github.com/swaggo/http-swagger"
	"log/slog"
	"net/http"
	"os"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response time for handler",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	cpuUsage = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "system_cpu_usage",
			Help: "Current CPU usage as a percentage",
		},
		getCPUUsage,
	)
	memoryUsage = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "system_memory_usage_bytes",
			Help: "Current memory usage in bytes",
		},
		getMemoryUsage,
	)
)

func main() {
	cfg := config.MustLoad()
	log := prettylogger.SetupLogger(cfg.Env)

	log.Info("Configuration loaded", slog.String("environment", cfg.Env))
	log.Info("Logger initialized", slog.String("level", cfg.Env))

	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(cpuUsage)
	prometheus.MustRegister(memoryUsage)

	log.Info("Prometheus metrics registered")

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

	router := setupRouter(cfg, log, chatClient, placesClient, charityClient, votesClient)
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

func setupRouter(cfg *config.Config, log *slog.Logger, chatClient proto.ChatServiceClient, placesClient proto.PlacesServiceClient, charityClient proto.CharityServiceClient, votesClient proto.VotesServiceClient) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(prometheusMiddleware)

	router.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		yamlFile, err := os.ReadFile("/root/swagger.yaml")
		if err != nil {
			http.Error(w, "Unable to read swagger.yaml", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/x-yaml")
		_, _ = w.Write(yamlFile)
	})

	router.Get("/api/docs/*", httpSwagger.Handler(
		httpSwagger.URL("http://95.174.92.20:8086/swagger"),
	),
	)

	router.Post("/api/chat/ask", chat.NewSendMessageHandler(log, chatClient))
	router.Post("/api/user/token", tokens.NewAddTokenHandler(log))

	router.Post("/api/places", places.NewGetPlacesHandler(log, placesClient))
	router.Get("/api/places/categories", places.NewGetCategoriesHandler(log, placesClient))
	router.Get("/api/places/tickets", places.NewGetTicketsHandler(log, placesClient))
	router.Post("/api/places/buy", places.NewBuyTicketHandler(log, placesClient))

	router.Get("/api/charity", charity.NewGetCollectionsHandler(log, charityClient))
	router.Get("/api/charity/categories", charity.NewGetCategoriesHandler(log, charityClient))
	router.Post("/api/charity/donate", charity.NewDonateHandler(log, charityClient))

	router.Get("/api/votes", votes.NewGetVotesHandler(log, votesClient))
	router.Get("/api/votes/categories", votes.NewGetCategoriesHandler(log, votesClient))
	router.Get("/api/votes/info", votes.NewGetVoteInfoHandler(log, votesClient))
	router.Post("/api/votes/rate", votes.NewVoteRateHandler(log, votesClient))
	router.Post("/api/votes/petition", votes.NewVotePetitionHandler(log, votesClient))
	router.Post("/api/votes/choice", votes.NewVoteChoiceHandler(log, votesClient))

	router.Handle("/metrics", promhttp.Handler())

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

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		excludedPaths := map[string]bool{
			"/api/docs/*": true,
			"/metrics":    true,
		}

		if excludedPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues(r.Method, r.URL.Path))
		defer timer.ObserveDuration()

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path).Inc()

		next.ServeHTTP(w, r)
	})
}

func getCPUUsage() float64 {
	percentages, err := cpu.Percent(0, false)
	if err != nil {
		return 0.0
	}
	return percentages[0] * 100
}

func getMemoryUsage() float64 {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0.0
	}
	return vmStat.UsedPercent
}
