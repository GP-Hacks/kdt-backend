package votes

import (
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
	"github.com/GP-Hacks/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"time"
)

type GetPetitionInfoResponseWithDefault struct {
	Response *PetitionInfoWithDefault `json:"response"`
}

type PetitionInfoWithDefault struct {
	ID           int            `json:"id"`
	Category     string         `json:"category"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Organization string         `json:"organization"`
	End          string         `json:"end"`
	Photo        string         `json:"photo"`
	Options      []string       `json:"options"`
	Stats        map[string]int `json:"stats"`
	Support      string         `json:"support,omitempty"`
}

func withDefaultPetitionInfo(resp *proto.GetPetitionInfoResponse) *GetPetitionInfoResponseWithDefault {
	stats := make(map[string]int)
	for k, v := range resp.GetResponse().Stats {
		stats[k] = int(v)
	}

	return &GetPetitionInfoResponseWithDefault{
		Response: &PetitionInfoWithDefault{
			ID:           int(resp.GetResponse().Id),
			Category:     resp.GetResponse().Category,
			Name:         resp.GetResponse().Name,
			Description:  resp.GetResponse().Description,
			Organization: resp.GetResponse().Organization,
			End:          resp.GetResponse().End.AsTime().Format(time.RFC3339),
			Photo:        resp.GetResponse().Photo,
			Options:      resp.GetResponse().Options,
			Stats:        stats,
			Support:      resp.GetResponse().Support,
		},
	}
}

func NewVotePetitionHandler(log *slog.Logger, votesClient proto.VotesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.votes.votePetition.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Received request to vote on petition")

		select {
		case <-ctx.Done():
			logger.Warn("Request cancelled by the client", slog.String("reason", ctx.Err().Error()))
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			logger.Warn("Missing authorization token")
			json.WriteError(w, http.StatusUnauthorized, "Authorization required")
			return
		}

		var request proto.VotePetitionRequest
		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Failed to parse JSON input", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if request.GetVoteId() == 0 {
			logger.Warn("Invalid or missing vote_id field", slog.Any("request", request))
			json.WriteError(w, http.StatusBadRequest, "Invalid vote_id field")
			return
		}

		if request.GetSupport() == "" {
			logger.Warn("Invalid or missing support field", slog.Any("request", request))
			json.WriteError(w, http.StatusBadRequest, "Invalid support field")
			return
		}

		request.Token = token

		resp, err := votesClient.VotePetition(ctx, &request)
		if err != nil {
			logger.Error("Failed to record vote", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not record vote")
			return
		}

		logger.Info("Vote recorded successfully", slog.Any("response", resp))
		json.WriteJSON(w, http.StatusOK, resp)
	}
}
