package votes

import (
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-commons/json"
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
		},
	}
}

func NewVotePetitionHandler(log *slog.Logger, votesClient proto.VotesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.votes.votePetition.New"
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

		var request proto.VotePetitionRequest
		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Invalid JSON input", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if request.GetVoteId() == 0 {
			logger.Warn("Invalid vote_id field")
			json.WriteError(w, http.StatusBadRequest, "Invalid vote_id field")
			return
		}

		if request.GetSupport() == "" {
			logger.Warn("Invalid support field")
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

		json.WriteJSON(w, http.StatusOK, resp)
		logger.Debug("Vote recorded successfully")
	}
}
