package votes

import (
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
	"github.com/GP-Hacks/kdt2024-commons/json"
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
	Rate         float64  `json:"rate,omitempty"`
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
			Rate:         float64(resp.GetResponse().Rate),
		},
	}
}

func NewVoteRateHandler(log *slog.Logger, votesClient proto.VotesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.votes.voteRate.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Received request to vote on rate")

		select {
		case <-ctx.Done():
			logger.Warn("Request cancelled by the client", slog.String("reason", ctx.Err().Error()))
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			logger.Warn("Authorization token is missing")
			json.WriteError(w, http.StatusUnauthorized, "Authorization token is required")
			return
		}

		var request proto.VoteRateRequest
		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Failed to parse JSON input", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if request.GetVoteId() == 0 {
			logger.Warn("Invalid or missing vote_id", slog.Any("request", request))
			json.WriteError(w, http.StatusBadRequest, "Invalid vote_id field")
			return
		}

		if request.GetRating() == 0 {
			logger.Warn("Invalid or missing rating", slog.Any("request", request))
			json.WriteError(w, http.StatusBadRequest, "Invalid rating field")
			return
		}

		request.Token = token

		resp, err := votesClient.VoteRate(ctx, &request)
		if err != nil {
			logger.Error("Failed to record vote", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Failed to record vote")
			return
		}

		logger.Info("Vote recorded successfully", slog.Any("response", resp))
		json.WriteJSON(w, http.StatusOK, resp)
	}
}
