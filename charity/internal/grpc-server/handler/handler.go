package handler

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/GP-Hacks/kdt2024-charity/config"
	"github.com/GP-Hacks/kdt2024-charity/internal/storage"
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
	"github.com/jackc/pgx/v5"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
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
	logger.Info("GRPCHandler initialized", slog.String("address", cfg.Address))
	return handler
}

func (h *GRPCHandler) GetCollections(ctx context.Context, request *proto.GetCollectionsRequest) (*proto.GetCollectionsResponse, error) {
	h.logger.Debug("Received GetCollections request", slog.Any("request", request))

	select {
	case <-ctx.Done():
		h.logger.Warn("GetCollections request was cancelled by client")
		return nil, status.Errorf(codes.Canceled, "Request was cancelled")
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
		return nil, h.handleStorageError(err, "collections")
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
			Photo:        collection.Photo,
		})
	}

	return &proto.GetCollectionsResponse{Response: responseCollections}, nil
}

func (h *GRPCHandler) GetCategories(ctx context.Context, request *proto.GetCategoriesRequest) (*proto.GetCategoriesResponse, error) {
	h.logger.Debug("Received GetCategories request", slog.Any("request", request))

	select {
	case <-ctx.Done():
		h.logger.Warn("GetCategories request was cancelled by client")
		return nil, status.Errorf(codes.Canceled, "Request was cancelled")
	default:
	}

	categories, err := h.storage.GetCategories(ctx)
	if err != nil {
		return nil, h.handleStorageError(err, "categories")
	}

	return &proto.GetCategoriesResponse{Categories: categories}, nil
}

func (h *GRPCHandler) Donate(ctx context.Context, request *proto.DonateRequest) (*proto.DonateResponse, error) {
	h.logger.Debug("Received Donate request", slog.Any("request", request))

	select {
	case <-ctx.Done():
		h.logger.Warn("Donate request was cancelled by client")
		return nil, status.Errorf(codes.Canceled, "Request was cancelled")
	default:
	}

	donationMessage := DonationMessage{
		UserToken:    request.GetToken(),
		CollectionID: int(request.GetCollectionId()),
		DonationTime: time.Now(),
		Amount:       int(request.GetAmount()),
	}

	h.logger.Info("Publishing donation to RabbitMQ", slog.String("queue_name", h.cfg.QueueName))
	if err := h.publishToRabbitMQ(donationMessage, h.cfg.QueueName); err != nil {
		h.logger.Error("Failed to publish donation to RabbitMQ", slog.Any("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "Failed to process donation, please try again later")
	}

	h.logger.Info("Updating collection in database", slog.Int("collection_id", donationMessage.CollectionID), slog.Int("amount", donationMessage.Amount))
	if err := h.storage.UpdateCollection(ctx, donationMessage.CollectionID, donationMessage.Amount); err != nil {
		h.logger.Error("Failed to update collection in database", slog.Any("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "Failed to process donation, please try again later")
	}

	h.logger.Info("Donation processed successfully", slog.String("user_token", donationMessage.UserToken), slog.Int("collection_id", donationMessage.CollectionID), slog.Int("amount", donationMessage.Amount))

	return &proto.DonateResponse{
		Response: "Thank you for your donation!",
	}, nil
}

func (h *GRPCHandler) HealthCheck(ctx context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	h.logger.Debug("Received HealthCheck request")

	h.logger.Info("HealthCheck passed")
	return &proto.HealthCheckResponse{IsHealthy: true}, nil
}

func (h *GRPCHandler) publishToRabbitMQ(message interface{}, queueName string) error {
	h.logger.Debug("Declaring RabbitMQ queue", slog.String("queue_name", queueName))
	q, err := h.mqch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		h.logger.Error("Failed to declare RabbitMQ queue", slog.String("queue_name", queueName), slog.Any("error", err.Error()))
		return err
	}

	h.logger.Debug("Marshalling message to JSON")
	body, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("Failed to marshal message to JSON", slog.Any("error", err.Error()))
		return err
	}

	h.logger.Debug("Publishing message to RabbitMQ", slog.String("queue_name", q.Name))
	err = h.mqch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		h.logger.Error("Failed to publish message to RabbitMQ", slog.String("queue_name", q.Name), slog.Any("error", err.Error()))
		return err
	}

	h.logger.Info("Message published to RabbitMQ successfully", slog.String("queue_name", q.Name))
	return nil
}

func (h *GRPCHandler) handleStorageError(err error, entity string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		h.logger.Warn("No records found in database", slog.String("entity", entity), slog.Any("error", err.Error()))
		return status.Errorf(codes.NotFound, "No %s found in database", entity)
	}
	h.logger.Error("Database operation failed", slog.String("entity", entity), slog.Any("error", err.Error()))
	return status.Errorf(codes.Internal, "Internal server error, please try again later")
}
