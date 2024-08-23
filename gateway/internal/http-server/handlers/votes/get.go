package votes

import (
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"time"
)

type GetVotesResponseWithDefault struct {
	Response []*VoteWithDefault `json:"response"`
}

type VoteWithDefault struct {
	ID           int      `json:"id"`
	Category     string   `json:"category"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Organization string   `json:"organization"`
	End          string   `json:"end"`
	Photo        string   `json:"photo"`
	Options      []string `json:"options"`
}

type VoteResponse struct {
	Response string `json:"response"`
}

func withDefaultVoteValues(resp *proto.GetVotesResponse) *GetVotesResponseWithDefault {
	def := &GetVotesResponseWithDefault{}
	for _, vote := range resp.GetResponse() {
		voteDef := &VoteWithDefault{
			ID:           int(vote.Id),
			Category:     vote.Category,
			Name:         vote.Name,
			Description:  vote.Description,
			Organization: vote.Organization,
			End:          vote.End.AsTime().Format(time.RFC3339),
			Photo:        vote.Photo,
			Options:      vote.Options,
		}
		if vote.Options == nil {
			voteDef.Options = []string{}
		}
		def.Response = append(def.Response, voteDef)
	}
	return def
}

func NewGetVotesHandler(log *slog.Logger, votesClient proto.VotesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.votes.get.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Received request to get all votes")

		select {
		case <-ctx.Done():
			logger.Warn("Request was cancelled by the client", slog.String("reason", ctx.Err().Error()))
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		resp, err := votesClient.GetVotes(ctx, &proto.GetVotesRequest{})
		if err != nil {
			logger.Error("Failed to retrieve votes", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Failed to retrieve votes")
			return
		}

		response := withDefaultVoteValues(resp)
		json.WriteJSON(w, http.StatusOK, response)
		logger.Debug("Votes retrieved successfully", slog.Any("response", response))
	}
}

func NewGetVoteInfoHandler(log *slog.Logger, votesClient proto.VotesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.votes.getVoteInfo.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Received request to get vote info")

		select {
		case <-ctx.Done():
			logger.Warn("Request was cancelled by the client", slog.String("reason", ctx.Err().Error()))
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		var request proto.GetVoteInfoRequest
		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Failed to parse JSON input", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		voteId := request.GetVoteId()
		if voteId == 0 {
			logger.Warn("Invalid vote_id field", slog.Any("request_payload", request))
			json.WriteError(w, http.StatusBadRequest, "Invalid vote_id field")
			return
		}

		votesResp, err := votesClient.GetVotes(ctx, &proto.GetVotesRequest{})
		if err != nil {
			logger.Error("Failed to retrieve votes", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Failed to retrieve votes")
			return
		}

		var voteResp *proto.Vote
		for _, vote := range votesResp.GetResponse() {
			if vote.Id == voteId {
				voteResp = vote
				break
			}
		}
		if voteResp == nil {
			logger.Warn("Vote not found", slog.Int("vote_id", int(voteId)))
			json.WriteError(w, http.StatusNotFound, "Vote not found")
			return
		}

		var detailedResp interface{}
		switch voteResp.Category {
		case "choice":
			choiceResp, err := votesClient.GetChoiceInfo(ctx, &proto.GetVoteInfoRequest{VoteId: voteId})
			if err != nil {
				logger.Error("Failed to retrieve choice info", slog.String("error", err.Error()))
				json.WriteError(w, http.StatusInternalServerError, "Failed to retrieve choice info")
				return
			}
			detailedResp = withDefaultChoiceInfo(choiceResp)
		case "petition":
			petitionResp, err := votesClient.GetPetitionInfo(ctx, &proto.GetVoteInfoRequest{VoteId: voteId})
			if err != nil {
				logger.Error("Failed to retrieve petition info", slog.String("error", err.Error()))
				json.WriteError(w, http.StatusInternalServerError, "Failed to retrieve petition info")
				return
			}
			detailedResp = withDefaultPetitionInfo(petitionResp)
		case "rate":
			rateResp, err := votesClient.GetRateInfo(ctx, &proto.GetVoteInfoRequest{VoteId: voteId})
			if err != nil {
				logger.Error("Failed to retrieve rate info", slog.String("error", err.Error()))
				json.WriteError(w, http.StatusInternalServerError, "Failed to retrieve rate info")
				return
			}
			detailedResp = withDefaultRateInfo(rateResp)
		default:
			logger.Warn("Unknown vote category", slog.String("category", voteResp.Category))
			json.WriteError(w, http.StatusBadRequest, "Unknown vote category")
			return
		}

		json.WriteJSON(w, http.StatusOK, detailedResp)
		logger.Debug("Vote info retrieved successfully", slog.Any("response", detailedResp))
	}
}
