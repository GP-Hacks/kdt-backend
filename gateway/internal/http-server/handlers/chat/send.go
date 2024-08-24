package chat

import (
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
	"github.com/GP-Hacks/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
)

func validateAuthorization(user string) (int, string) {
	if user == "" {
		return http.StatusUnauthorized, "Authorization token is required"
	}
	return http.StatusOK, ""
}

func validateSendMessageRequest(request *proto.SendMessageRequest) (int, string) {
	messages := request.GetMessages()
	if len(messages) != 1 {
		return http.StatusBadRequest, "Request should contain exactly one message"
	}

	message := messages[0]
	if message.GetContent() == "" {
		return http.StatusBadRequest, "Message content cannot be empty"
	}
	if message.GetRole() != "user" {
		return http.StatusBadRequest, "Message role must be 'user'"
	}

	return http.StatusOK, ""
}

func NewSendMessageHandler(log *slog.Logger, chatClient proto.ChatServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.chat.send.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Handling request to send message")

		select {
		case <-ctx.Done():
			logger.Warn("Request was cancelled by the client")
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		user := r.Header.Get("Authorization")
		statusCode, message := validateAuthorization(user)
		if statusCode != http.StatusOK {
			logger.Warn("Authorization validation failed", slog.String("message", message))
			json.WriteError(w, statusCode, message)
			return
		}

		var request proto.SendMessageRequest
		if err := json.ReadJSON(r, &request); err != nil {
			logger.Error("Failed to parse JSON input", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		statusCode, message = validateSendMessageRequest(&request)
		if statusCode != http.StatusOK {
			logger.Warn("SendMessageRequest validation failed", slog.String("message", message))
			json.WriteError(w, statusCode, message)
			return
		}

		resp, err := chatClient.SendMessage(ctx, &proto.SendMessageRequest{Messages: request.GetMessages()})
		if err != nil {
			logger.Error("gRPC SendMessage call failed", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Failed to send message")
			return
		}

		logger.Debug("Message sent successfully", slog.Any("response", resp))
		json.WriteJSON(w, http.StatusOK, resp)
	}
}
