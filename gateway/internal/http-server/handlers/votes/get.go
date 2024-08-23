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
		logger := log.With(slog.String("op", op), slog.Any("request_id", reqID), slog.Any("ip", r.RemoteAddr))

		select {
		case <-ctx.Done():
			logger.Warn("Request cancelled by the client")
			return
		default:
		}

		resp, err := votesClient.GetVotes(ctx, &proto.GetVotesRequest{})
		if err != nil {
			logger.Error("Failed to get votes", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not get votes")
			return
		}

		json.WriteJSON(w, http.StatusOK, withDefaultVoteValues(resp))
		logger.Debug("Votes retrieved successfully")
	}
}

func NewGetVoteInfoHandler(log *slog.Logger, votesClient proto.VotesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.votes.getVoteInfo.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(slog.String("op", op), slog.Any("request_id", reqID), slog.Any("ip", r.RemoteAddr))

		select {
		case <-ctx.Done():
			logger.Warn("Request cancelled by the client")
			return
		default:
		}

		var request proto.GetVoteInfoRequest
		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Invalid JSON input", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}
		voteId := request.GetVoteId()
		if voteId == 0 {
			logger.Warn("Invalid vote_id field")
			json.WriteError(w, http.StatusBadRequest, "Invalid vote_id field")
			return
		}

		votesResp, err := votesClient.GetVotes(ctx, &proto.GetVotesRequest{})
		if err != nil {
			logger.Error("Failed to get vote info", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not get vote info")
			return
		}
		var voteResp *proto.Vote
		for _, vote := range votesResp.GetResponse() {
			if vote.Id == voteId {
				voteResp = vote
			}
		}
		if voteResp == nil {
			json.WriteError(w, http.StatusNotFound, "Vote not found")
			return
		}

		category := voteResp.Category
		var detailedResp interface{}
		switch category {
		case "choice":
			choiceResp, err := votesClient.GetChoiceInfo(ctx, &proto.GetVoteInfoRequest{VoteId: voteId})
			if err != nil {
				logger.Error("Failed to get choice info", slog.String("error", err.Error()))
				json.WriteError(w, http.StatusInternalServerError, "Could not get choice info")
				return
			}
			detailedResp = withDefaultChoiceInfo(choiceResp)
		case "petition":
			petitionResp, err := votesClient.GetPetitionInfo(ctx, &proto.GetVoteInfoRequest{VoteId: voteId})
			if err != nil {
				logger.Error("Failed to get petition info", slog.String("error", err.Error()))
				json.WriteError(w, http.StatusInternalServerError, "Could not get petition info")
				return
			}
			detailedResp = withDefaultPetitionInfo(petitionResp)
		case "rate":
			rateResp, err := votesClient.GetRateInfo(ctx, &proto.GetVoteInfoRequest{VoteId: voteId})
			if err != nil {
				logger.Error("Failed to get rate info", slog.String("error", err.Error()))
				json.WriteError(w, http.StatusInternalServerError, "Could not get rate info")
				return
			}
			detailedResp = withDefaultRateInfo(rateResp)
		default:
			logger.Warn("Unknown vote category")
			json.WriteError(w, http.StatusBadRequest, "Unknown vote category")
			return
		}
		json.WriteJSON(w, http.StatusOK, detailedResp)
		logger.Debug("Vote info retrieved successfully")
		return
	}
}
