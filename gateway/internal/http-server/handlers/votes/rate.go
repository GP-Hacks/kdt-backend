package votes

import (
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"time"
)

type GetRateInfoResponseWithDefault struct {
	Response *RateInfoWithDefault `json:"response"`
}

type RateInfoWithDefault struct {
	ID           int      `json:"id"`
	Category     string   `json:"category"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Organization string   `json:"organization"`
	End          string   `json:"end"`
	Photo        string   `json:"photo"`
	Options      []string `json:"options"`
	Mid          float64  `json:"mid"`
}

func withDefaultRateInfo(resp *proto.GetRateInfoResponse) *GetRateInfoResponseWithDefault {
	return &GetRateInfoResponseWithDefault{
		Response: &RateInfoWithDefault{
			ID:           int(resp.GetResponse().Id),
			Category:     resp.GetResponse().Category,
			Name:         resp.GetResponse().Name,
			Description:  resp.GetResponse().Description,
			Organization: resp.GetResponse().Organization,
			End:          resp.GetResponse().End.AsTime().Format(time.RFC3339),
			Photo:        resp.GetResponse().Photo,
			Options:      resp.GetResponse().Options,
			Mid:          float64(resp.GetResponse().Mid),
		},
	}
}

func NewVoteRateHandler(log *slog.Logger, votesClient proto.VotesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.votes.voteRate.New"
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

		var request proto.VoteRateRequest
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

		if request.GetRating() == 0 {
			logger.Warn("Invalid rating field")
			json.WriteError(w, http.StatusBadRequest, "Invalid rating field")
			return
		}

		request.Token = token

		resp, err := votesClient.VoteRate(ctx, &request)
		if err != nil {
			logger.Error("Failed to record vote", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not record vote")
			return
		}

		json.WriteJSON(w, http.StatusOK, resp)
		logger.Debug("Vote recorded successfully")
	}
}
