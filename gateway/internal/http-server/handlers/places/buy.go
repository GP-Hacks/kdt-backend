package places

import (
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log/slog"
	"net/http"
	"time"
)

func NewBuyTicketHandler(log *slog.Logger, placesClient proto.PlacesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.places.buy.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(slog.String("op", op), slog.Any("request_id", reqID), slog.Any("ip", r.RemoteAddr))

		select {
		case <-ctx.Done():
			logger.Warn("Request cancelled by the client")
			return
		default:
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			json.WriteError(w, http.StatusUnauthorized, "Authorization required")
			return
		}

		var request struct {
			PlaceId   int       `json:"place_id"`
			Timestamp time.Time `json:"timestamp"`
		}

		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Failed to read JSON", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if request.PlaceId == 0 {
			logger.Warn("Invalid place_id field")
			json.WriteError(w, http.StatusBadRequest, "Invalid place_id field")
			return
		}

		if request.Timestamp.IsZero() {
			logger.Warn("Invalid timestamp field")
			json.WriteError(w, http.StatusBadRequest, "Invalid timestamp field")
			return
		}

		protoRequest := &proto.BuyTicketRequest{
			Token:     token,
			PlaceId:   int32(request.PlaceId),
			Timestamp: timestamppb.New(request.Timestamp),
		}

		resp, err := placesClient.BuyTicket(ctx, protoRequest)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				json.WriteError(w, http.StatusNotFound, "Place not found")
				logger.Warn("Place not found", slog.Int("id", int(protoRequest.GetPlaceId())))
				return
			}
			json.WriteError(w, http.StatusInternalServerError, "Could not save your order")
			logger.Error("Failed to save order", slog.String("error", err.Error()))
			return
		}

		json.WriteJSON(w, http.StatusOK, resp)
		logger.Debug("Ticket bought successfully", slog.Any("response", resp))
	}
}
