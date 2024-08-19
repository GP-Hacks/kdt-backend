package handler

import (
	"context"
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-places/internal/storage"
	"google.golang.org/grpc"
	"log/slog"
	"math"
	"sort"
)

func distance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Радиус Земли в километрах
	lat1 = lat1 * math.Pi / 180
	lon1 = lon1 * math.Pi / 180
	lat2 = lat2 * math.Pi / 180
	lon2 = lon2 * math.Pi / 180

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

type GRPCHandler struct {
	proto.UnimplementedPlacesServiceServer
	storage *storage.PostgresStorage
	logger  *slog.Logger
}

func NewGRPCHandler(server *grpc.Server, storage *storage.PostgresStorage, logger *slog.Logger) *GRPCHandler {
	handler := &GRPCHandler{storage: storage, logger: logger}
	proto.RegisterPlacesServiceServer(server, handler)
	return handler
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
		if err != nil {
			h.logger.Error("Failed to get places", slog.Any("error", err.Error()))
			return nil, err
		}
	} else {
		places, err = h.storage.GetPlacesByCategory(ctx, category)
		if err != nil {
			h.logger.Error("Failed to get places", slog.Any("error", err.Error()))
			return nil, err
		}
	}

	userLatitude := request.GetLatitude()
	userLongitude := request.GetLongitude()

	sort.Slice(places, func(i, j int) bool {
		return distance(places[i].Latitude, places[i].Longitude, userLatitude, userLongitude) < distance(places[j].Latitude, places[j].Longitude, userLatitude, userLongitude)
	})

	var responsePlaces []*proto.Place
	for _, place := range places {
		placePhotos, err := h.storage.GetPhotosById(ctx, place.ID)
		if err != nil {
			h.logger.Error("Failed to get places photos", slog.Any("error", err.Error()))
			return nil, err
		}
		if placePhotos == nil {
			placePhotos = []*storage.Photo{}
		}
		var protoPhotos []*proto.Photo
		for _, placePhoto := range placePhotos {
			protoPhotos = append(protoPhotos, &proto.Photo{Url: placePhoto.Url})
		}
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
			Photos:      protoPhotos,
		})
	}

	return &proto.GetPlacesResponse{Response: responsePlaces}, nil
}

func (h *GRPCHandler) HealthCheck(ctx context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	h.logger.Debug("Processing HealthCheck")
	return &proto.HealthCheckResponse{
		IsHealthy: true,
	}, nil
}
