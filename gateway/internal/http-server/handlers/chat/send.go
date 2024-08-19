package chat

import (
	"github.com/GP-Hack/kdt2024-commons/api/proto"
	"github.com/GP-Hack/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
)

func NewSendMessageHandler(log *slog.Logger, chatClient proto.ChatServiceClient) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.chat.send.New"
		ctx := r.Context()
		log = log.With(slog.String("op", op), slog.Any("request_id", middleware.GetReqID(r.Context())), slog.Any("ip", r.RemoteAddr))

		select {
		case <-ctx.Done():
			log.Warn("Request cancelled by the client")
			return
		default:
		}

		user := r.Header.Get("Authorization")
		if user == "" {
			json.WriteError(w, http.StatusUnauthorized, "Authorization required")
		}
		var request *proto.SendMessageRequest
		if err := json.ReadJSON(r, &request); err != nil {
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			log.Error("Failed to read JSON", slog.String("error", err.Error()))
			return
		}
		// validation
		messages := request.GetMessages()
		if len(messages) != 1 {
			json.WriteError(w, http.StatusBadRequest, "Invalid messages count")
			log.Warn("Invalid messages count")
			return
		}
		message := messages[0]
		if message.GetContent() == "" {
			json.WriteError(w, http.StatusBadRequest, "Invalid message content")
			log.Warn("Invalid message content")
			return
		}
		if message.GetRole() != "user" {
			json.WriteError(w, http.StatusBadRequest, "Invalid message role")
			log.Warn("Invalid message role")
			return
		}

		resp, err := chatClient.SendMessage(ctx, &proto.SendMessageRequest{
			Messages: messages,
		})

		if err != nil {
			json.WriteError(w, http.StatusInternalServerError, "Failed to send message")
			log.Error("gRPC SendMessage call failed", slog.String("error", err.Error()))
			return
		}

		json.WriteJSON(w, http.StatusOK, resp)
		log.Debug("Message sent successfully", slog.Any("response", resp))
	})
}
