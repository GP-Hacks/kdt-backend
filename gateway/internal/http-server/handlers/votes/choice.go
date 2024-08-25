package votes

import (
	"fmt"
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
	"github.com/GP-Hacks/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"net/http"
	"time"
)

type GetChoiceInfoResponseWithDefault struct {
	Response *ChoiceInfoWithDefault `json:"response"`
}

type ChoiceInfoWithDefault struct {
	ID           int            `json:"id"`
	Category     string         `json:"category"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Organization string         `json:"organization"`
	End          string         `json:"end"`
	Photo        string         `json:"photo"`
	Options      []string       `json:"options"`
	Stats        map[string]int `json:"stats"`
	Choice       string         `json:"choice,omitempty"`
}

func withDefaultChoiceInfo(resp *proto.GetChoiceInfoResponse) *GetChoiceInfoResponseWithDefault {
	stats := make(map[string]int)
	for _, opt := range resp.GetResponse().Options {
		stats[opt] = int(resp.GetResponse().Stats[opt])
	}

	return &GetChoiceInfoResponseWithDefault{
		Response: &ChoiceInfoWithDefault{
			ID:           int(resp.GetResponse().Id),
			Category:     resp.GetResponse().Category,
			Name:         resp.GetResponse().Name,
			Description:  resp.GetResponse().Description,
			Organization: resp.GetResponse().Organization,
			End:          resp.GetResponse().End.AsTime().Format(time.RFC3339),
			Photo:        resp.GetResponse().Photo,
			Options:      resp.GetResponse().Options,
			Stats:        stats,
			Choice:       resp.GetResponse().Choice,
		},
	}
}

func NewVoteChoiceHandler(log *slog.Logger, votesClient proto.VotesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.votes.voteChoice.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Processing vote choice request")

		select {
		case <-ctx.Done():
			logger.Warn("Request was cancelled by the client", slog.String("reason", ctx.Err().Error()))
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			logger.Warn("Missing Authorization header")
			json.WriteError(w, http.StatusUnauthorized, "Authorization required")
			return
		}

		var request proto.VoteChoiceRequest
		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Failed to parse JSON input", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if request.GetVoteId() == 0 {
			logger.Warn("Invalid vote_id field in request", slog.String("request_payload", fmt.Sprintf("%+v", request)))
			json.WriteError(w, http.StatusBadRequest, "Invalid vote_id field")
			return
		}

		if request.GetChoice() == "" {
			logger.Warn("Invalid choice field in request", slog.String("request_payload", fmt.Sprintf("%+v", request)))
			json.WriteError(w, http.StatusBadRequest, "Invalid choice field")
			return
		}

		request.Token = token

		_, err := votesClient.VoteChoice(ctx, &request)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				logger.Warn("Vote choice not found", slog.String("error", err.Error()), slog.Any("request", request))
				json.WriteError(w, http.StatusNotFound, "Choice not found")
				return
			}
			logger.Error("Failed to record vote", slog.String("error", err.Error()), slog.Any("request", request))
			json.WriteError(w, http.StatusInternalServerError, "Could not record vote")
			return
		}

		response := map[string]string{"response": "Vote recorded successfully"}
		logger.Info("Vote recorded successfully", slog.Any("response", response))
		json.WriteJSON(w, http.StatusOK, response)
	}
}
