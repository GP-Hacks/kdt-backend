package chat

import (
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
)

func validateAuthorization(user string) (int, string) {
	if user == "" {
		return http.StatusUnauthorized, "Authorization required"
	}
	return http.StatusOK, ""
}

func validateSendMessageRequest(request *proto.SendMessageRequest) (int, string) {
	messages := request.GetMessages()
	if len(messages) != 1 {
		return http.StatusBadRequest, "Invalid messages count"
	}

	message := messages[0]
	if message.GetContent() == "" {
		return http.StatusBadRequest, "Invalid message content"
	}
	if message.GetRole() != "user" {
		return http.StatusBadRequest, "Invalid message role"
	}

	return http.StatusOK, ""
}

func NewSendMessageHandler(log *slog.Logger, chatClient proto.ChatServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.chat.send.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(slog.String("op", op), slog.Any("request_id", reqID), slog.Any("ip", r.RemoteAddr))

		select {
		case <-ctx.Done():
			logger.Warn("Request cancelled by the client")
			return
		default:
		}

		user := r.Header.Get("Authorization")
		statusCode, message := validateAuthorization(user)
		if statusCode != http.StatusOK {
			json.WriteError(w, statusCode, message)
			return
		}

		var request proto.SendMessageRequest
		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Failed to read JSON", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		statusCode, message = validateSendMessageRequest(&request)
		if statusCode != http.StatusOK {
			logger.Warn(message)
			json.WriteError(w, statusCode, message)
			return
		}

		resp, err := chatClient.SendMessage(ctx, &proto.SendMessageRequest{Messages: request.GetMessages()})
		if err != nil {
			logger.Error("gRPC SendMessage call failed", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Failed to send message")
			return
		}

		json.WriteJSON(w, http.StatusOK, resp)
		logger.Debug("Message sent successfully", slog.Any("response", resp))
	}
}
