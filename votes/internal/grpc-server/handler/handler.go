package handler

import (
	"context"
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
	"github.com/GP-Hacks/kdt2024-votes/config"
	"github.com/GP-Hacks/kdt2024-votes/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log/slog"
)

type GRPCHandler struct {
	cfg *config.Config
	proto.UnimplementedVotesServiceServer
	storage *storage.PostgresStorage
	logger  *slog.Logger
}

func NewGRPCHandler(cfg *config.Config, server *grpc.Server, storage *storage.PostgresStorage, logger *slog.Logger) *GRPCHandler {
	handler := &GRPCHandler{cfg: cfg, storage: storage, logger: logger}
	proto.RegisterVotesServiceServer(server, handler)
	logger.Info("GRPCHandler initialized", slog.String("address", cfg.Address))
	return handler
}

func (h *GRPCHandler) GetVotes(ctx context.Context, request *proto.GetVotesRequest) (*proto.GetVotesResponse, error) {
	h.logger.Debug("Received GetVotes request", slog.Any("request", request))

	select {
	case <-ctx.Done():
		h.logger.Warn("GetVotes request was cancelled by client")
		return nil, status.Errorf(codes.Canceled, "Request was cancelled")
	default:
	}

	var votes []*storage.Vote
	var err error

	category := request.GetCategory()
	if category == "all" {
		votes, err = h.storage.GetVotes(ctx)
	} else {
		votes, err = h.storage.GetVotesByCategory(ctx, category)
	}
	if err != nil {
		return nil, h.handleStorageError(err, "votes")
	}

	var protoVotes []*proto.Vote
	for _, vote := range votes {
		protoVotes = append(protoVotes, &proto.Vote{
			Id:           int32(vote.ID),
			Category:     vote.Category,
			Name:         vote.Name,
			Description:  vote.Description,
			Organization: vote.Organization,
			End:          timestamppb.New(vote.EndTime),
			Photo:        vote.Photo,
			Options:      vote.Options,
		})
	}

	return &proto.GetVotesResponse{Response: protoVotes}, nil
}

func (h *GRPCHandler) GetCategories(ctx context.Context, request *proto.GetCategoriesRequest) (*proto.GetCategoriesResponse, error) {
	h.logger.Debug("Received GetCategories request", slog.Any("request", request))

	select {
	case <-ctx.Done():
		h.logger.Warn("GetCategories request was cancelled by client")
		return nil, status.Errorf(codes.Canceled, "Request was cancelled")
	default:
	}

	categories, err := h.storage.GetCategories(ctx)
	if err != nil {
		return nil, h.handleStorageError(err, "categories")
	}

	return &proto.GetCategoriesResponse{Categories: categories}, nil
}

func (h *GRPCHandler) GetRateInfo(ctx context.Context, request *proto.GetVoteInfoRequest) (*proto.GetRateInfoResponse, error) {
	h.logger.Debug("Received GetRateInfo request", slog.Any("request", request))

	select {
	case <-ctx.Done():
		h.logger.Warn("GetRateInfo request was cancelled by client")
		return nil, status.Errorf(codes.Canceled, "Request was cancelled")
	default:
	}

	rates, err := h.storage.GetUserRates(ctx, request.Token)
	if err != nil {
		return nil, h.handleStorageError(err, "rates")
	}
	var urate *storage.UserRate
	if len(rates) != 0 {
		for _, rate := range rates {
			if int32(rate.ID) == request.VoteId {
				urate = rate
			}
		}
	}
	var xxx float32
	if urate != nil {
		xxx = float32(urate.Rate)
	}

	rateInfo, err := h.storage.GetRateInfo(ctx, int(request.VoteId))
	if err != nil {
		return nil, h.handleStorageError(err, "fetching rate info")
	}

	return &proto.GetRateInfoResponse{
		Response: &proto.VoteInfo{
			Id:           int32(rateInfo.ID),
			Category:     rateInfo.Category,
			Name:         rateInfo.Name,
			Description:  rateInfo.Description,
			Organization: rateInfo.Organization,
			End:          timestamppb.New(rateInfo.EndTime),
			Options:      rateInfo.Options,
			Photo:        rateInfo.Photo,
			Mid:          float32(rateInfo.Mid),
			Rate:         xxx,
		},
	}, nil
}

func (h *GRPCHandler) GetPetitionInfo(ctx context.Context, request *proto.GetVoteInfoRequest) (*proto.GetPetitionInfoResponse, error) {
	h.logger.Debug("Received GetPetitionInfo request", slog.Any("request", request))

	petitions, err := h.storage.GetUserPetitions(ctx, request.Token)
	if err != nil {
		return nil, h.handleStorageError(err, "petitions")
	}
	var upet *storage.UserPetition
	if len(petitions) != 0 {
		for _, pet := range petitions {
			if int32(pet.ID) == request.VoteId {
				upet = pet
			}
		}
	}

	var xxx string
	if upet != nil {
		xxx = upet.Support
	}

	petitionInfo, err := h.storage.GetPetitionInfo(ctx, int(request.VoteId))
	if err != nil {
		return nil, h.handleStorageError(err, "fetching petition info")
	}

	return &proto.GetPetitionInfoResponse{
		Response: &proto.PetitionInfo{
			Id:           int32(petitionInfo.ID),
			Category:     petitionInfo.Category,
			Name:         petitionInfo.Name,
			Description:  petitionInfo.Description,
			Organization: petitionInfo.Organization,
			End:          timestamppb.New(petitionInfo.EndTime),
			Options:      petitionInfo.Options,
			Photo:        petitionInfo.Photo,
			Stats:        petitionInfo.Stats,
			Support:      xxx,
		},
	}, nil
}

func (h *GRPCHandler) GetChoiceInfo(ctx context.Context, request *proto.GetVoteInfoRequest) (*proto.GetChoiceInfoResponse, error) {
	h.logger.Debug("Received GetChoiceInfo request", slog.Any("request", request))

	choices, err := h.storage.GetUserChoices(ctx, request.Token)

	if err != nil {
		return nil, h.handleStorageError(err, "choices")
	}
	var uchoice *storage.UserChoice
	if len(choices) != 0 {
		for _, choice := range choices {
			if int32(choice.ID) == request.VoteId {
				uchoice = choice
			}
		}
	}

	var xxx string
	if uchoice != nil {
		xxx = uchoice.Choice
	}

	choiceInfo, err := h.storage.GetChoiceInfo(ctx, int(request.VoteId))
	if err != nil {
		return nil, h.handleStorageError(err, "fetching choice info")
	}

	return &proto.GetChoiceInfoResponse{
		Response: &proto.ChoiceInfo{
			Id:           int32(choiceInfo.ID),
			Category:     choiceInfo.Category,
			Name:         choiceInfo.Name,
			Description:  choiceInfo.Description,
			Organization: choiceInfo.Organization,
			End:          timestamppb.New(choiceInfo.EndTime),
			Options:      choiceInfo.Options,
			Photo:        choiceInfo.Photo,
			Stats:        choiceInfo.Stats,
			Choice:       xxx,
		},
	}, nil
}

func (h *GRPCHandler) VoteRate(ctx context.Context, request *proto.VoteRateRequest) (*proto.VoteResponse, error) {
	h.logger.Debug("Received VoteRate request", slog.Any("request", request))

	err := h.storage.VoteRate(ctx, request.Token, int(request.VoteId), int(request.Rating))
	if err != nil {
		return nil, h.handleStorageError(err, "voting rate")
	}

	h.logger.Info("Successfully recorded rate vote", slog.String("token", request.Token), slog.Int("vote_id", int(request.VoteId)))
	return &proto.VoteResponse{Response: "Vote recorded successfully"}, nil
}

func (h *GRPCHandler) VotePetition(ctx context.Context, request *proto.VotePetitionRequest) (*proto.VoteResponse, error) {
	h.logger.Debug("Received VotePetition request", slog.Any("request", request))

	err := h.storage.VotePetition(ctx, request.Token, int(request.VoteId), request.Support)
	if err != nil {
		return nil, h.handleStorageError(err, "voting petition")
	}

	h.logger.Info("Successfully recorded petition vote", slog.String("token", request.Token), slog.Int("vote_id", int(request.VoteId)))
	return &proto.VoteResponse{Response: "Vote recorded successfully"}, nil
}

func (h *GRPCHandler) VoteChoice(ctx context.Context, request *proto.VoteChoiceRequest) (*proto.VoteResponse, error) {
	h.logger.Debug("Received VoteChoice request", slog.Any("request", request))

	err := h.storage.VoteChoice(ctx, request.Token, int(request.VoteId), request.Choice)
	if err != nil {
		return nil, h.handleStorageError(err, "voting choice")
	}

	h.logger.Info("Successfully recorded choice vote", slog.String("token", request.Token), slog.Int("vote_id", int(request.VoteId)))
	return &proto.VoteResponse{Response: "Vote recorded successfully"}, nil
}

func (h *GRPCHandler) HealthCheck(ctx context.Context, request *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	h.logger.Debug("Received HealthCheck request")

	h.logger.Info("HealthCheck passed")
	return &proto.HealthCheckResponse{IsHealthy: true}, nil
}

func (h *GRPCHandler) handleStorageError(err error, context string) error {
	h.logger.Error("Storage operation failed", slog.String("context", context), slog.String("error", err.Error()))
	return status.Errorf(codes.Internal, "Failed to process %s: %v", context, err)
}
