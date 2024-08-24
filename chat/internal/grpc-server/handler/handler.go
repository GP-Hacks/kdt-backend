package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/GP-Hacks/kdt2024-chat/internal/storage"
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
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
	h.logger.Debug("Received SendMessage request", slog.Any("request", req))

	select {
	case <-ctx.Done():
		h.logger.Warn("SendMessage request was cancelled by client")
		return nil, status.Errorf(codes.Canceled, "Request was cancelled")
	default:
	}

	message := req.GetMessages()[0].GetContent()
	redisKey := "chatbot:" + message

	h.logger.Debug("Checking cache for response", slog.String("redis_key", redisKey))
	cachedResponse, err := h.getCachedResponse(ctx, redisKey)
	if err == nil && cachedResponse != "" {
		h.logger.Info("Cache hit: returning cached response", slog.String("redis_key", redisKey))
		return &proto.SendMessageResponse{Response: cachedResponse}, nil
	}

	h.logger.Debug("Cache miss: sending request to bot", slog.String("message", message))
	response, err := h.fetchResponseFromBot(ctx, message)
	if err != nil {
		h.logger.Error("Failed to fetch response from bot", slog.String("error", err.Error()))
		return nil, err
	}

	h.logger.Debug("Caching bot response", slog.String("redis_key", redisKey), slog.String("response", response))
	if err := h.cacheResponse(ctx, redisKey, response); err != nil {
		h.logger.Error("Failed to cache bot response", slog.String("error", err.Error()), slog.String("redis_key", redisKey))
	}

	h.logger.Info("Successfully processed SendMessage request", slog.String("response", response))
	return &proto.SendMessageResponse{Response: response}, nil
}

func (h *GRPCHandler) HealthCheck(ctx context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	h.logger.Debug("Received HealthCheck request")

	h.logger.Info("HealthCheck passed")
	return &proto.HealthCheckResponse{IsHealthy: true}, nil
}

func (h *GRPCHandler) getCachedResponse(ctx context.Context, key string) (string, error) {
	h.logger.Debug("Fetching response from Redis", slog.String("redis_key", key))
	response, err := h.storage.Get(ctx, key)
	if err != nil {
		h.logger.Error("Failed to fetch response from Redis", slog.String("redis_key", key), slog.String("error", err.Error()))
		return "", err
	}
	return response, nil
}

func (h *GRPCHandler) fetchResponseFromBot(ctx context.Context, message string) (string, error) {
	postData := BotRequest{
		Messages: []BotMessage{
			{Role: "user", Content: message},
		},
	}
	jsonData, err := json.Marshal(postData)
	if err != nil {
		h.logger.Error("Failed to marshal bot request data", slog.String("error", err.Error()), slog.Any("postData", postData))
		return "", status.Errorf(codes.Internal, "Unable to process your request at this time, please try again later")
	}

	httpResp, err := h.sendHTTPRequest(ctx, jsonData)
	if err != nil {
		h.logger.Error("Failed to send HTTP request to bot", slog.String("error", err.Error()))
		return "", err
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			h.logger.Warn("Failed to close bot response body", slog.String("error", err.Error()))
		}
	}()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		h.logger.Error("Failed to read bot response body", slog.String("error", err.Error()))
		return "", status.Errorf(codes.Internal, "Unable to read bot response, please try again later")
	}

	h.logger.Debug("Bot response received", slog.String("response", string(body)))
	return string(body), nil
}

func (h *GRPCHandler) sendHTTPRequest(ctx context.Context, jsonData []byte) (*http.Response, error) {
	h.logger.Debug("Creating HTTP request to bot", slog.String("url", botURL))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", botURL, bytes.NewBuffer(jsonData))
	if err != nil {
		h.logger.Error("Failed to create HTTP request", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "Unable to create request to the bot, please try again later")
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	h.logger.Debug("Sending HTTP request to bot", slog.String("url", botURL))
	httpResp, err := client.Do(httpReq)
	if err != nil {
		h.logger.Error("Failed to send HTTP request", slog.String("error", err.Error()), slog.String("url", botURL))
		return nil, status.Errorf(codes.Internal, "Failed to contact the bot service, please try again later")
	}

	return httpResp, nil
}

func (h *GRPCHandler) cacheResponse(ctx context.Context, key, response string) error {
	h.logger.Debug("Caching response in Redis", slog.String("redis_key", key), slog.String("response", response))
	if err := h.storage.Set(ctx, key, response, cacheDuration); err != nil {
		h.logger.Error("Failed to cache response in Redis", slog.String("redis_key", key), slog.String("error", err.Error()))
		return err
	}
	return nil
}
