package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-places/config"
	"github.com/GP-Hack/kdt2024-places/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"math"
	"sort"
	"time"
)

type NotificationMessage struct {
	UserID  string    `json:"user_id"`
	Header  string    `json:"header"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

type PurchaseMessage struct {
	User         string    `json:"user"`
	PlaceID      int       `json:"place_id"`
	EventTime    time.Time `json:"event_time"`
	PurchaseTime time.Time `json:"purchase_time"`
	Cost         int       `json:"cost"`
}

const EarthRadius = 6371

func distance(lat1, lon1, lat2, lon2 float64) float64 {
	lat1, lon1 = toRadians(lat1), toRadians(lon1)
	lat2, lon2 = toRadians(lat2), toRadians(lon2)

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadius * c
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180
}

func roundMinutes(t time.Time) time.Time {
	minute := t.Minute()
	roundedMinute := 5 * ((minute + 4) / 5)

	if roundedMinute == 60 {
		roundedMinute = 0
		t = t.Add(time.Hour)
	}

	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), roundedMinute, 0, 0, t.Location())
}

type GRPCHandler struct {
	cfg *config.Config
	proto.UnimplementedPlacesServiceServer
	storage *storage.PostgresStorage
	logger  *slog.Logger
	mqch    *amqp.Channel
}

func NewGRPCHandler(cfg *config.Config, server *grpc.Server, storage *storage.PostgresStorage, logger *slog.Logger, mqch *amqp.Channel) *GRPCHandler {
	handler := &GRPCHandler{cfg: cfg, storage: storage, logger: logger, mqch: mqch}
	proto.RegisterPlacesServiceServer(server, handler)
	return handler
}

func (h *GRPCHandler) handleStorageError(err error, entity string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		h.logger.Error("No "+entity+" in DB", slog.Any("error", err.Error()))
		return status.Errorf(codes.NotFound, "No such "+entity+" in database")
	}
	h.logger.Error("Failed to get "+entity, slog.Any("error", err.Error()))
	return status.Errorf(codes.Internal, "Please try again later")
}

func (h *GRPCHandler) GetPlaces(ctx context.Context, request *proto.GetPlacesRequest) (*proto.GetPlacesResponse, error) {
	h.logger.Debug("Processing GetPlaces")

	select {
	case <-ctx.Done():
		h.logger.Warn("Request was cancelled")
		return nil, ctx.Err()
	default:
	}

	var places []*storage.Place
	var err error

	category := request.GetCategory()
	if category == "all" {
		places, err = h.storage.GetPlaces(ctx)
	} else {
		places, err = h.storage.GetPlacesByCategory(ctx, category)
	}
	if err != nil {
		return nil, h.handleStorageError(err, "places")
	}

	userLatitude := request.GetLatitude()
	userLongitude := request.GetLongitude()

	sort.Slice(places, func(i, j int) bool {
		return distance(places[i].Latitude, places[i].Longitude, userLatitude, userLongitude) <
			distance(places[j].Latitude, places[j].Longitude, userLatitude, userLongitude)
	})

	var responsePlaces []*proto.Place
	for _, place := range places {
		protoPhotos, err := h.getPlacePhotos(ctx, place.ID)
		if err != nil {
			return nil, err
		}

		times := []string{place.Time, roundMinutes(time.Now()).Format("15:04")}
		sort.Strings(times)

		responsePlaces = append(responsePlaces, &proto.Place{
			Id:          int32(place.ID),
			Category:    place.Category,
			Description: place.Description,
			Latitude:    place.Latitude,
			Longitude:   place.Longitude,
			Location:    place.Location,
			Name:        place.Name,
			Tel:         place.Tel,
			Website:     place.Website,
			Cost:        int32(place.Cost),
			Times:       times,
			Photos:      protoPhotos,
		})
	}

	return &proto.GetPlacesResponse{Response: responsePlaces}, nil
}

func (h *GRPCHandler) getPlacePhotos(ctx context.Context, placeID int) ([]*proto.Photo, error) {
	placePhotos, err := h.storage.GetPhotosById(ctx, placeID)
	if err != nil {
		return nil, h.handleStorageError(err, "photos")
	}
	if placePhotos == nil {
		placePhotos = []*storage.Photo{}
	}

	var protoPhotos []*proto.Photo
	for _, placePhoto := range placePhotos {
		protoPhotos = append(protoPhotos, &proto.Photo{Url: placePhoto.Url})
	}

	return protoPhotos, nil
}

func (h *GRPCHandler) BuyTicket(ctx context.Context, request *proto.BuyTicketRequest) (*proto.BuyTicketResponse, error) {
	h.logger.Debug("Processing BuyTicket")
	select {
	case <-ctx.Done():
		h.logger.Warn("Request was cancelled")
		return nil, ctx.Err()
	default:
	}

	dbPlace, err := h.storage.GetPlaceById(ctx, int(request.GetPlaceId()))
	if err != nil {
		return nil, h.handleStorageError(err, "place")
	}

	err = h.storage.SaveOrder(ctx, request.GetToken(), int(request.GetPlaceId()), request.GetTimestamp().AsTime(), dbPlace.Cost)
	if err != nil {
		h.logger.Error("Failed to save order", slog.Any("error", err.Error()))
		return &proto.BuyTicketResponse{
			Response: "Failed to buy ticket",
		}, status.Errorf(codes.Internal, "Please try again later")
	}

	message := NotificationMessage{
		UserID:  request.GetToken(),
		Header:  "Напоминание о покупке!",
		Content: fmt.Sprintf("Вы приобрели билет на %s в %s", dbPlace.Name, request.GetTimestamp().AsTime().Format("15:04")),
		Time:    request.GetTimestamp().AsTime().Add(-15 * time.Minute),
	}
	err = h.publishToRabbitMQ(message, h.cfg.QueueNotifications)
	if err != nil {
		h.logger.Error("Failed to publish message to RabbitMQ", slog.Any("error", err.Error()))
	}

	purchaseMessage := PurchaseMessage{
		User:         request.GetToken(),
		PlaceID:      dbPlace.ID,
		EventTime:    request.GetTimestamp().AsTime(),
		PurchaseTime: time.Now(),
		Cost:         dbPlace.Cost,
	}
	err = h.publishToRabbitMQ(purchaseMessage, h.cfg.QueuePurchases)
	if err != nil {
		h.logger.Error("Failed to publish message to RabbitMQ", slog.Any("error", err.Error()))
	}

	return &proto.BuyTicketResponse{
		Response: "Successfully bought ticket",
	}, nil
}

func (h *GRPCHandler) GetCategories(ctx context.Context, request *proto.GetCategoriesRequest) (*proto.GetCategoriesResponse, error) {
	h.logger.Debug("Processing GetCategories")

	categories, err := h.storage.GetCategories(ctx)
	if err != nil {
		return nil, h.handleStorageError(err, "categories")
	}

	return &proto.GetCategoriesResponse{Categories: categories}, nil
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
