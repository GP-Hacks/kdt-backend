package places

import (
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
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
	Photos      []*proto.Photo `json:"photos"`
}

func withDefaultValues(resp *proto.GetPlacesResponse) *GetPlacesResponseWithDefault {
	def := &GetPlacesResponseWithDefault{}
	response := resp.GetResponse()
	for _, place := range response {
		placeDef := &PlaceWithDefault{}
		placeDef.ID = int(place.Id)
		placeDef.Category = place.Category
		placeDef.Description = place.Description
		placeDef.Latitude = place.Latitude
		placeDef.Longitude = place.Longitude
		placeDef.Location = place.Location
		placeDef.Name = place.Name
		placeDef.Tel = place.Tel
		placeDef.Website = place.Website
		if place.Photos == nil {
			place.Photos = []*proto.Photo{}
		}
		placeDef.Photos = place.Photos
		def.Response = append(def.Response, placeDef)
	}
	return def
}

func NewGetPlacesHandler(log *slog.Logger, placesClient proto.PlacesServiceClient) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.places.get.New"
		ctx := r.Context()
		log = log.With(slog.String("op", op), slog.Any("request_id", middleware.GetReqID(r.Context())), slog.Any("ip", r.RemoteAddr))

		select {
		case <-ctx.Done():
			log.Warn("Request cancelled by the client")
			return
		default:
		}

		var request *proto.GetPlacesRequest
		if err := json.ReadJSON(r, &request); err != nil {
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			log.Error("Failed to read JSON", slog.String("error", err.Error()))
			return
		}
		// validation
		category := request.GetCategory()
		if category == "" {
			json.WriteError(w, http.StatusBadRequest, "Invalid category field")
			log.Warn("Invalid category field")
			return
		}
		latitude := request.GetLatitude()
		if latitude == 0 {
			json.WriteError(w, http.StatusBadRequest, "Invalid latitude field")
			log.Warn("Invalid latitude field")
			return
		}
		longitude := request.GetLongitude()
		if longitude == 0 {
			json.WriteError(w, http.StatusBadRequest, "Invalid longitude field")
			log.Warn("Invalid longitude field")
			return
		}

		resp, err := placesClient.GetPlaces(ctx, &proto.GetPlacesRequest{
			Category:  category,
			Latitude:  latitude,
			Longitude: longitude,
		})

		if err != nil {
			//if errors.Is(err, pgx.ErrNoRows) {
			json.WriteError(w, http.StatusNotFound, "Places by this category not found")
			log.Warn("Places by this category not found", slog.String("category", category))
			return
			//}
			/*json.WriteError(w, http.StatusInternalServerError, "Failed to get places")
			log.Error("gRPC GetPlaces call failed", slog.String("error", err.Error()))
			return*/
		}

		json.WriteJSON(w, http.StatusOK, withDefaultValues(resp))
		log.Debug("Places got successfully", slog.Any("response", resp))
	})
}
