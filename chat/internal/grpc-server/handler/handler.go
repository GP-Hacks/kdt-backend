package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/GP-Hack/kdt2024-chat/internal/storage"
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"google.golang.org/grpc"
	"io"
	"log/slog"
	"net/http"
	"time"
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
	h.logger.Info("Processing SendMessage")

	botURL := "https://app.fastbots.ai/api/bots/clzydq0yf01hpr4beei5nl8xd/ask"
	message := req.GetMessages()[0].GetContent()
	redisKey := "chatbot:" + message

	cachedResponse, err := h.storage.Get(ctx, redisKey)
	if err == nil {
		h.logger.Debug("Cache found, returning cached response")
		return &proto.SendMessageResponse{
			Response: cachedResponse,
		}, nil
	}

	postData := BotRequest{Messages: []BotMessage{{
		Role:    "user",
		Content: message,
	}}}

	jsonData, err := json.Marshal(postData)
	if err != nil {
		h.logger.Error("Failed to marshal postData", slog.String("error", err.Error()))
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", botURL, bytes.NewBuffer(jsonData))
	if err != nil {
		h.logger.Error("Failed to create http request to bot", slog.String("error", err.Error()))
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		h.logger.Error("Failed to send http request to bot", slog.String("error", err.Error()))
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(httpResp.Body)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		h.logger.Error("Failed to read bot response body", slog.String("error", err.Error()))
		return nil, err
	}

	err = h.storage.Set(ctx, redisKey, string(body), 72*time.Hour)
	if err != nil {
		h.logger.Error("Failed to save response in Redis", slog.String("error", err.Error()))
	}

	return &proto.SendMessageResponse{
		Response: string(body),
	}, nil
}
