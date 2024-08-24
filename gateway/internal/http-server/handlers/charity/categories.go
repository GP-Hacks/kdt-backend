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

func NewGetCategoriesHandler(log *slog.Logger, charityClient proto.CharityServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.charity.getcategories.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("op", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Processing request")

		select {
		case <-ctx.Done():
			logger.Warn("Request cancelled by the client")
			json.WriteError(w, http.StatusRequestTimeout, "Request timed out")
			return
		default:
		}

		req := &proto.GetCategoriesRequest{}
		logger.Debug("Sending request to charity service", slog.Any("request", req))

		resp, err := charityClient.GetCategories(ctx, req)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				logger.Warn("No categories found", slog.String("error", err.Error()))
				json.WriteError(w, http.StatusNotFound, "No categories found")
				return
			}
			logger.Error("Failed to retrieve categories from charity service", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not retrieve categories")
			return
		}

		response := struct {
			Categories []string `json:"categories"`
		}{
			Categories: resp.GetCategories(),
		}

		logger.Debug("Successfully retrieved categories", slog.Int("count", len(response.Categories)))
		json.WriteJSON(w, http.StatusOK, response)
	}
}
