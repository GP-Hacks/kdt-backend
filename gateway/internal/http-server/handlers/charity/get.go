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

type GetCollectionsResponseWithDefault struct {
	Response []*CollectionWithDefault `json:"response"`
}

type CollectionWithDefault struct {
	ID           int    `json:"id"`
	Category     string `json:"category"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Organization string `json:"organization"`
	Phone        string `json:"phone"`
	Website      string `json:"website"`
	Goal         int    `json:"goal"`
	Current      int    `json:"current"`
	Photo        string `json:"photo"`
}

func withDefaultValues(resp *proto.GetCollectionsResponse) *GetCollectionsResponseWithDefault {
	def := &GetCollectionsResponseWithDefault{}
	for _, collection := range resp.GetResponse() {
		collectionDef := &CollectionWithDefault{
			ID:           int(collection.Id),
			Category:     collection.Category,
			Name:         collection.Name,
			Description:  collection.Description,
			Organization: collection.Organization,
			Phone:        collection.Phone,
			Website:      collection.Website,
			Goal:         int(collection.Goal),
			Current:      int(collection.Current),
			Photo:        collection.Photo,
		}
		def.Response = append(def.Response, collectionDef)
	}
	return def
}

func NewGetCollectionsHandler(log *slog.Logger, charityClient proto.CharityServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.charity.getCollections.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Received request to get collections")

		select {
		case <-ctx.Done():
			logger.Warn("Request was cancelled by the client")
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

		request := proto.GetCollectionsRequest{Category: category}

		logger.Info("Fetching collections for category", slog.String("category", request.GetCategory()))

		resp, err := charityClient.GetCollections(ctx, &request)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				logger.Warn("Collections not found for category", slog.String("category", request.GetCategory()))
				json.WriteError(w, http.StatusNotFound, "Collections not found")
				return
			}
			logger.Error("Failed to fetch collections", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not retrieve collections")
			return
		}

		logger.Debug("Successfully retrieved collections", slog.Int("num_collections", len(resp.GetResponse())))

		json.WriteJSON(w, http.StatusOK, withDefaultValues(resp))
	}
}
