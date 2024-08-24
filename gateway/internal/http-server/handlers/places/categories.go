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

func NewGetCategoriesHandler(log *slog.Logger, placesClient proto.PlacesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.places.getcategories.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Processing request to get categories")

		select {
		case <-ctx.Done():
			logger.Warn("Request was cancelled by the client", slog.String("reason", ctx.Err().Error()))
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		req := &proto.GetCategoriesRequest{}
		logger.Debug("Sending request to get categories", slog.Any("request", req))

		resp, err := placesClient.GetCategories(ctx, req)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				logger.Warn("No categories found in response", slog.String("error", err.Error()))
				json.WriteError(w, http.StatusNotFound, "No categories found")
				return
			}
			logger.Error("Failed to retrieve categories from gRPC server", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not retrieve categories")
			return
		}

		response := struct {
			Response []string `json:"response"`
		}{
			Response: resp.GetCategories(),
		}

		logger.Debug("Categories successfully retrieved", slog.Any("response", response))
		json.WriteJSON(w, http.StatusOK, response)
	}
}
