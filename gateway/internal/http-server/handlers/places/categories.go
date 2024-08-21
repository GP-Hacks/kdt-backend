package places

import (
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"net/http"
)

func NewGetCategoriesHandler(log *slog.Logger, placesClient proto.PlacesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.places.getcategories.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(slog.String("op", op), slog.Any("request_id", reqID), slog.Any("ip", r.RemoteAddr))

		select {
		case <-ctx.Done():
			logger.Warn("Request cancelled by the client")
			return
		default:
		}

		req := &proto.GetCategoriesRequest{}

		resp, err := placesClient.GetCategories(ctx, req)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				logger.Warn("No categories found")
				json.WriteError(w, http.StatusNotFound, "No categories found")
				return
			}
			logger.Error("Failed to get categories", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not retrieve categories")
			return
		}

		response := struct {
			Response []string `json:"response"`
		}{
			Response: resp.GetCategories(),
		}

		json.WriteJSON(w, http.StatusOK, response)
		logger.Debug("Categories retrieved successfully")
	}
}
