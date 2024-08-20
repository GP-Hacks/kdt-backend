package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/GP-Hack/kdt2024-chat/internal/storage"
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	botURL        = "https://app.fastbots.ai/api/bots/clzydq0yf01hpr4beei5nl8xd/ask"
	cacheDuration = 72 * time.Hour
)

type BotRequest struct {
	Messages []BotMessage `json:"messages"`
}

type BotMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GRPCHandler struct {
	proto.UnimplementedChatServiceServer
	storage *storage.RedisStorage
	logger  *slog.Logger
}

func NewGRPCHandler(server *grpc.Server, storage *storage.RedisStorage, logger *slog.Logger) *GRPCHandler {
	handler := &GRPCHandler{storage: storage, logger: logger}
	proto.RegisterChatServiceServer(server, handler)
	return handler
}

func (h *GRPCHandler) SendMessage(ctx context.Context, req *proto.SendMessageRequest) (*proto.SendMessageResponse, error) {
	h.logger.Debug("Processing SendMessage", slog.Any("proto request", req))

	select {
	case <-ctx.Done():
		h.logger.Warn("Request was cancelled")
		return nil, ctx.Err()
	default:
	}

	message := req.GetMessages()[0].GetContent()
	redisKey := "chatbot:" + message

	cachedResponse, err := h.getCachedResponse(ctx, redisKey)
	if err == nil {
		h.logger.Debug("Cache found, returning cached response")
		return &proto.SendMessageResponse{Response: cachedResponse}, nil
	}

	response, err := h.fetchResponseFromBot(ctx, message)
	if err != nil {
		return nil, err
	}

	if err := h.cacheResponse(ctx, redisKey, response); err != nil {
		h.logger.Error("Failed to save response in Redis", slog.String("error", err.Error()))
	}

	return &proto.SendMessageResponse{Response: response}, nil
}

func (h *GRPCHandler) HealthCheck(ctx context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	h.logger.Debug("Processing HealthCheck")
	return &proto.HealthCheckResponse{IsHealthy: true}, nil
}

func (h *GRPCHandler) getCachedResponse(ctx context.Context, key string) (string, error) {
	return h.storage.Get(ctx, key)
}

func (h *GRPCHandler) fetchResponseFromBot(ctx context.Context, message string) (string, error) {
	postData := BotRequest{Messages: []BotMessage{{Role: "user", Content: message}}}
	jsonData, err := json.Marshal(postData)
	if err != nil {
		h.logger.Error("Failed to marshal postData", slog.String("error", err.Error()))
		return "", status.Errorf(codes.Internal, "Please try again later")
	}

	httpResp, err := h.sendHTTPRequest(ctx, jsonData)
	if err != nil {
		return "", err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		h.logger.Error("Failed to read bot response body", slog.String("error", err.Error()))
		return "", status.Errorf(codes.Internal, "Please try again later")
	}

	return string(body), nil
}

func (h *GRPCHandler) sendHTTPRequest(ctx context.Context, jsonData []byte) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", botURL, bytes.NewBuffer(jsonData))
	if err != nil {
		h.logger.Error("Failed to create http request to bot", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "Please try again later")
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(httpReq)
}

func (h *GRPCHandler) cacheResponse(ctx context.Context, key, response string) error {
	return h.storage.Set(ctx, key, response, cacheDuration)
}
