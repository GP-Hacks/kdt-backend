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
		}
		def.Response = append(def.Response, collectionDef)
	}
	return def
}

func NewGetCollectionsHandler(log *slog.Logger, charityClient proto.CharityServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.charity.get.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(slog.String("op", op), slog.Any("request_id", reqID), slog.Any("ip", r.RemoteAddr))

		select {
		case <-ctx.Done():
			logger.Warn("Request cancelled by the client")
			return
		default:
		}

		var request proto.GetCollectionsRequest
		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Invalid JSON input", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if request.GetCategory() == "" {
			logger.Warn("Request does not contain a category")
			json.WriteError(w, http.StatusBadRequest, "Invalid category field")
			return
		}

		resp, err := charityClient.GetCollections(ctx, &request)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				logger.Warn("Collections not found", slog.String("category", request.GetCategory()))
				json.WriteError(w, http.StatusNotFound, "Collections not found")
				return
			}
			logger.Error("Failed to get collections", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not get collections")
			return
		}

		json.WriteJSON(w, http.StatusOK, withDefaultValues(resp))
		logger.Debug("Collections retrieved successfully")
	}
}
