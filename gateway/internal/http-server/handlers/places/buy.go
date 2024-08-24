package places

import (
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
	"github.com/GP-Hacks/kdt2024-commons/json"
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
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Processing buy ticket request")

		select {
		case <-ctx.Done():
			logger.Warn("Request was cancelled by the client", slog.String("reason", ctx.Err().Error()))
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			logger.Warn("Authorization header is missing or empty")
			json.WriteError(w, http.StatusUnauthorized, "Authorization required")
			return
		}

		var request struct {
			PlaceId   int       `json:"place_id"`
			Timestamp time.Time `json:"timestamp"`
		}
		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Failed to parse JSON input", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if request.PlaceId <= 0 {
			logger.Warn("Invalid place_id field", slog.Int("place_id", request.PlaceId))
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
				logger.Warn("Place not found", slog.Int("place_id", int(protoRequest.GetPlaceId())))
				json.WriteError(w, http.StatusNotFound, "Place not found")
				return
			}
			logger.Error("Failed to process buy ticket request", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not save your order")
			return
		}

		json.WriteJSON(w, http.StatusOK, resp)
		logger.Debug("Ticket successfully bought", slog.Any("response", resp))
	}
}
