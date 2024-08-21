package charity

import (
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-commons/json"
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
			CollectionId int `json:"collection_id"`
			Amount       int `json:"amount"`
		}

		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Failed to read JSON", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if request.CollectionId == 0 {
			logger.Warn("Invalid collection_id field")
			json.WriteError(w, http.StatusBadRequest, "Invalid collection_id field")
			return
		}

		if request.Amount <= 0 {
			logger.Warn("Invalid amount field")
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
				json.WriteError(w, http.StatusNotFound, "Collection not found")
				logger.Warn("Collection not found", slog.Int("id", int(protoRequest.GetCollectionId())))
				return
			}
			json.WriteError(w, http.StatusInternalServerError, "Could not save your donation")
			logger.Error("Failed to save donation", slog.String("error", err.Error()))
			return
		}

		json.WriteJSON(w, http.StatusOK, resp)
		logger.Debug("Donation sent successfully", slog.Any("response", resp))
	}
}
