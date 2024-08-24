package places

import (
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
	"github.com/GP-Hacks/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"net/http"
)

type GetPlacesResponseWithDefault struct {
	Response []*PlaceWithDefault `json:"response"`
}

type PlaceWithDefault struct {
	ID          int            `json:"id"`
	Category    string         `json:"category"`
	Description string         `json:"description"`
	Latitude    float64        `json:"latitude"`
	Longitude   float64        `json:"longitude"`
	Location    string         `json:"location"`
	Name        string         `json:"name"`
	Tel         string         `json:"tel"`
	Website     string         `json:"website"`
	Cost        int            `json:"cost"`
	Times       []string       `json:"times"`
	Photos      []*proto.Photo `json:"photos"`
}

func withDefaultValues(resp *proto.GetPlacesResponse) *GetPlacesResponseWithDefault {
	def := &GetPlacesResponseWithDefault{}
	for _, place := range resp.GetResponse() {
		placeDef := &PlaceWithDefault{
			ID:          int(place.Id),
			Category:    place.Category,
			Description: place.Description,
			Latitude:    place.Latitude,
			Longitude:   place.Longitude,
			Location:    place.Location,
			Name:        place.Name,
			Tel:         place.Tel,
			Website:     place.Website,
			Cost:        int(place.Cost),
			Times:       place.Times,
			Photos:      place.Photos,
		}
		if place.Photos == nil {
			placeDef.Photos = []*proto.Photo{}
		}
		def.Response = append(def.Response, placeDef)
	}
	return def
}

func NewGetPlacesHandler(log *slog.Logger, placesClient proto.PlacesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.places.get.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Processing request to get places")

		select {
		case <-ctx.Done():
			logger.Warn("Request was cancelled by the client", slog.String("reason", ctx.Err().Error()))
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		category := r.URL.Query().Get("category")

		if category == "" {
			logger.Warn("Invalid category parameter")
			json.WriteError(w, http.StatusBadRequest, "Invalid category parameter")
			return
		}

		request := proto.GetPlacesRequest{Category: category}

		resp, err := placesClient.GetPlaces(ctx, &request)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				logger.Warn("No places found for the given criteria", slog.String("category", request.GetCategory()), slog.Float64("latitude", request.GetLatitude()), slog.Float64("longitude", request.GetLongitude()))
				json.WriteError(w, http.StatusNotFound, "No places found for the given criteria")
				return
			}
			logger.Error("Failed to retrieve places from gRPC service", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not retrieve places")
			return
		}

		response := withDefaultValues(resp)
		logger.Debug("Places successfully retrieved", slog.Any("response", response))
		json.WriteJSON(w, http.StatusOK, response)
	}
}
