package handler

import (
	"context"
	"github.com/GP-Hack/kdt2024-commons/api/proto"
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
	return handler
}

func (h *GRPCHandler) GetVotes(ctx context.Context, request *proto.GetVotesRequest) (*proto.GetVotesResponse, error) {
	h.logger.Debug("Processing GetVotes")

	votes, err := h.storage.GetVotes(ctx)
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

func (h *GRPCHandler) GetRateInfo(ctx context.Context, request *proto.GetVoteInfoRequest) (*proto.GetRateInfoResponse, error) {
	h.logger.Debug("Processing GetRateInfo")

	rateInfo, err := h.storage.GetRateInfo(ctx, int(request.VoteId))
	if err != nil {
		return nil, h.handleStorageError(err, "rate info")
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
		},
	}, nil
}

func (h *GRPCHandler) GetPetitionInfo(ctx context.Context, request *proto.GetVoteInfoRequest) (*proto.GetPetitionInfoResponse, error) {
	h.logger.Debug("Processing GetPetitionInfo")

	petitionInfo, err := h.storage.GetPetitionInfo(ctx, int(request.VoteId))
	if err != nil {
		return nil, h.handleStorageError(err, "petition info")
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
		},
	}, nil
}

func (h *GRPCHandler) GetChoiceInfo(ctx context.Context, request *proto.GetVoteInfoRequest) (*proto.GetChoiceInfoResponse, error) {
	h.logger.Debug("Processing GetChoiceInfo")

	choiceInfo, err := h.storage.GetChoiceInfo(ctx, int(request.VoteId))
	if err != nil {
		return nil, h.handleStorageError(err, "choice info")
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
		},
	}, nil
}

func (h *GRPCHandler) VoteRate(ctx context.Context, request *proto.VoteRateRequest) (*proto.VoteResponse, error) {
	h.logger.Debug("Processing VoteRate")

	err := h.storage.VoteRate(ctx, request.Token, int(request.VoteId), int(request.Rating))
	if err != nil {
		return nil, h.handleStorageError(err, "voting rate")
	}

	return &proto.VoteResponse{Response: "Vote recorded successfully"}, nil
}

func (h *GRPCHandler) VotePetition(ctx context.Context, request *proto.VotePetitionRequest) (*proto.VoteResponse, error) {
	h.logger.Debug("Processing VotePetition")

	err := h.storage.VotePetition(ctx, request.Token, int(request.VoteId), request.Support)
	if err != nil {
		return nil, h.handleStorageError(err, "voting petition")
	}

	return &proto.VoteResponse{Response: "Vote recorded successfully"}, nil
}

func (h *GRPCHandler) VoteChoice(ctx context.Context, request *proto.VoteChoiceRequest) (*proto.VoteResponse, error) {
	h.logger.Debug("Processing VoteChoice")

	err := h.storage.VoteChoice(ctx, request.Token, int(request.VoteId), request.Choice)
	if err != nil {
		return nil, h.handleStorageError(err, "voting choice")
	}

	return &proto.VoteResponse{Response: "Vote recorded successfully"}, nil
}

func (h *GRPCHandler) HealthCheck(ctx context.Context, request *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	h.logger.Debug("Processing HealthCheck")
	return &proto.HealthCheckResponse{
		IsHealthy: true,
	}, nil
}

func (h *GRPCHandler) handleStorageError(err error, context string) error {
	h.logger.Error("Storage error", slog.String("context", context), slog.String("error", err.Error()))
	return status.Errorf(codes.Internal, "failed to process %s", context)
}
