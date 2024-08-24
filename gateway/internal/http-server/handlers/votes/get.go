package votes

import (
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
	"github.com/GP-Hacks/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"net/http"
	"strconv"
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

		category := r.URL.Query().Get("category")

		if category == "" {
			logger.Warn("Request missing category")
			json.WriteError(w, http.StatusBadRequest, "Category field is required")
			return
		}

		resp, err := votesClient.GetVotes(ctx, &proto.GetVotesRequest{Category: category})
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

		reqVoteId := r.URL.Query().Get("vote_id")

		if reqVoteId == "" {
			logger.Warn("Request missing vote_id")
			json.WriteError(w, http.StatusBadRequest, "vote_id field is required")
			return
		}

		voteId, err := strconv.Atoi(reqVoteId)

		if err != nil {
			logger.Warn("Request bad vote_id")
			json.WriteError(w, http.StatusBadRequest, "vote_id field is NaN")
			return
		}

		if voteId == 0 {
			logger.Warn("Invalid vote_id field")
			json.WriteError(w, http.StatusBadRequest, "Invalid vote_id field")
			return
		}

		votesResp, err := votesClient.GetVotes(ctx, &proto.GetVotesRequest{Category: "all"})
		if err != nil {
			logger.Error("Failed to retrieve votes", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Failed to retrieve votes")
			return
		}
		var voteResp *proto.Vote
		for _, vote := range votesResp.GetResponse() {
			if vote.Id == int32(voteId) {
				voteResp = vote
				break
			}
		}

		token := r.Header.Get("Authorization")

		if voteResp == nil {
			logger.Warn("Vote not found", slog.Int("vote_id", int(voteId)))
			json.WriteError(w, http.StatusNotFound, "Vote not found")
			return
		}

		var detailedResp interface{}
		switch voteResp.Category {
		case "choice":
			choiceResp, err := votesClient.GetChoiceInfo(ctx, &proto.GetVoteInfoRequest{VoteId: int32(voteId), Token: token})
			if err != nil {
				logger.Error("Failed to retrieve choice info", slog.String("error", err.Error()))
				json.WriteError(w, http.StatusInternalServerError, "Failed to retrieve choice info")
				return
			}
			detailedResp = withDefaultChoiceInfo(choiceResp)
		case "petition":
			petitionResp, err := votesClient.GetPetitionInfo(ctx, &proto.GetVoteInfoRequest{VoteId: int32(voteId), Token: token})
			if err != nil {
				logger.Error("Failed to retrieve petition info", slog.String("error", err.Error()))
				json.WriteError(w, http.StatusInternalServerError, "Failed to retrieve petition info")
				return
			}
			detailedResp = withDefaultPetitionInfo(petitionResp)
		case "rate":
			rateResp, err := votesClient.GetRateInfo(ctx, &proto.GetVoteInfoRequest{VoteId: int32(voteId), Token: token})
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

func NewGetCategoriesHandler(log *slog.Logger, votesClient proto.VotesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.votes.getcategories.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Processing request to get categories")

		select {
		case <-ctx.Done():
			logger.Warn("Request was cancelled by the client", slog.String("reason", ctx.Err().Error()))
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		req := &proto.GetCategoriesRequest{}
		logger.Debug("Sending request to get categories", slog.Any("request", req))

		resp, err := votesClient.GetCategories(ctx, req)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				logger.Warn("No categories found in response", slog.String("error", err.Error()))
				json.WriteError(w, http.StatusNotFound, "No categories found")
				return
			}
			logger.Error("Failed to retrieve categories from gRPC server", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not retrieve categories")
			return
		}

		response := struct {
			Response []string `json:"response"`
		}{
			Response: resp.GetCategories(),
		}

		logger.Debug("Categories successfully retrieved", slog.Any("response", response))
		json.WriteJSON(w, http.StatusOK, response)
	}
}
