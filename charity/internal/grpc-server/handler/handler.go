package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hacks/kdt2024-charity/config"
	"github.com/GP-Hacks/kdt2024-charity/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"time"
)

type DonationMessage struct {
	UserToken    string    `json:"user_token"`
	CollectionID int       `json:"collection_id"`
	DonationTime time.Time `json:"donation_time"`
	Amount       int       `json:"amount"`
}

type GRPCHandler struct {
	cfg *config.Config
	proto.UnimplementedCharityServiceServer
	storage *storage.PostgresStorage
	logger  *slog.Logger
	mqch    *amqp.Channel
}

func NewGRPCHandler(cfg *config.Config, server *grpc.Server, storage *storage.PostgresStorage, logger *slog.Logger, mqch *amqp.Channel) *GRPCHandler {
	handler := &GRPCHandler{cfg: cfg, storage: storage, logger: logger, mqch: mqch}
	proto.RegisterCharityServiceServer(server, handler)
	return handler
}

func (h *GRPCHandler) GetCollections(ctx context.Context, request *proto.GetCollectionsRequest) (*proto.GetCollectionsResponse, error) {
	h.logger.Debug("Processing GetCollections")

	select {
	case <-ctx.Done():
		h.logger.Warn("Request was cancelled")
		return nil, ctx.Err()
	default:
	}

	var collections []*storage.Collection
	var err error

	category := request.GetCategory()
	if category == "all" {
		collections, err = h.storage.GetCollections(ctx)
	} else {
		collections, err = h.storage.GetCollectionsByCategory(ctx, category)
	}
	if err != nil {
		return nil, h.handleStorageError(err, "places")
	}

	var responseCollections []*proto.Collection
	for _, collection := range collections {
		responseCollections = append(responseCollections, &proto.Collection{
			Id:           int32(collection.ID),
			Category:     collection.Category,
			Name:         collection.Name,
			Description:  collection.Description,
			Organization: collection.Organization,
			Phone:        collection.Phone,
			Website:      collection.Website,
			Goal:         int32(collection.Goal),
			Current:      int32(collection.Current),
		})
	}

	return &proto.GetCollectionsResponse{Response: responseCollections}, nil
}

func (h *GRPCHandler) GetCategories(ctx context.Context, request *proto.GetCategoriesRequest) (*proto.GetCategoriesResponse, error) {
	h.logger.Debug("Processing GetCategories")
	categories, err := h.storage.GetCategories(ctx)
	if err != nil {
		return nil, h.handleStorageError(err, "categories")
	}

	return &proto.GetCategoriesResponse{Categories: categories}, nil
}

func (h *GRPCHandler) Donate(ctx context.Context, request *proto.DonateRequest) (*proto.DonateResponse, error) {
	h.logger.Debug("Processing Donate")
	select {
	case <-ctx.Done():
		h.logger.Warn("Request was cancelled")
		return nil, ctx.Err()
	default:
	}
	donationMessage := DonationMessage{
		UserToken:    request.GetToken(),
		CollectionID: int(request.GetCollectionId()),
		DonationTime: time.Now(),
		Amount:       int(request.GetAmount()),
	}
	err := h.publishToRabbitMQ(donationMessage, h.cfg.QueueName)
	if err != nil {
		h.logger.Error("Failed to publish message to RabbitMQ", slog.Any("error", err.Error()))
	}

	return &proto.DonateResponse{
		Response: "Successfully donated",
	}, nil
}

func (h *GRPCHandler) HealthCheck(ctx context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	h.logger.Debug("Processing HealthCheck")
	return &proto.HealthCheckResponse{
		IsHealthy: true,
	}, nil
}

func (h *GRPCHandler) publishToRabbitMQ(message interface{}, queueName string) error {
	q, err := h.mqch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		h.logger.Error("Failed to declare a queue", slog.Any("error", err.Error()))
		return err
	}
	body, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("Failed to marshal message to JSON", slog.Any("error", err.Error()))
		return err
	}
	err = h.mqch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		})
	if err != nil {
		h.logger.Error("Failed to publish a message", slog.Any("error", err.Error()))
		return err
	}
	return nil
}

func (h *GRPCHandler) handleStorageError(err error, entity string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		h.logger.Error("No "+entity+" in DB", slog.Any("error", err.Error()))
		return status.Errorf(codes.NotFound, "No such "+entity+" in database")
	}
	h.logger.Error("Failed to get "+entity, slog.Any("error", err.Error()))
	return status.Errorf(codes.Internal, "Please try again later")
}
