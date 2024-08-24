package charity

import (
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
	"github.com/GP-Hacks/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"net/http"
)

func NewDonateHandler(log *slog.Logger, charityClient proto.CharityServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.charity.donate.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Processing donation request")

		select {
		case <-ctx.Done():
			logger.Warn("Request was cancelled by the client")
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			logger.Warn("Authorization header missing")
			json.WriteError(w, http.StatusUnauthorized, "Authorization required")
			return
		}

		var request struct {
			CollectionId int `json:"collection_id"`
			Amount       int `json:"amount"`
		}

		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Failed to parse JSON request", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if request.CollectionId <= 0 {
			logger.Warn("Invalid collection_id field", slog.Int("collection_id", request.CollectionId))
			json.WriteError(w, http.StatusBadRequest, "Invalid collection_id field")
			return
		}

		if request.Amount <= 0 {
			logger.Warn("Invalid amount field", slog.Int("amount", request.Amount))
			json.WriteError(w, http.StatusBadRequest, "Invalid amount field")
			return
		}

		protoRequest := &proto.DonateRequest{
			Token:        token,
			CollectionId: int32(request.CollectionId),
			Amount:       int32(request.Amount),
		}

		resp, err := charityClient.Donate(ctx, protoRequest)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				logger.Warn("Collection not found for donation", slog.Int("collection_id", request.CollectionId))
				json.WriteError(w, http.StatusNotFound, "Collection not found")
				return
			}
			logger.Error("Failed to process donation", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not save your donation")
			return
		}

		logger.Info("Donation processed successfully", slog.Any("response", resp))
		json.WriteJSON(w, http.StatusOK, resp)
	}
}
