package tokens

import (
	"github.com/GP-Hacks/kdt2024-commons/json"
	"github.com/GP-Hacks/kdt2024-gateway/internal/storage"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
)

type TokenRequest struct {
	Token string `json:"token"`
}

func NewAddTokenHandler(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.tokens.add.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Processing request to add token")
		select {
		case <-ctx.Done():
			logger.Warn("Request was cancelled by the client", slog.String("reason", ctx.Err().Error()))
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			logger.Warn("Authorization header is missing")
			json.WriteError(w, http.StatusUnauthorized, "Authorization required")
			return
		}

		var tokenReq TokenRequest
		if err := json.ReadJSON(r, &tokenReq); err != nil {
			logger.Error("Failed to parse JSON request", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if tokenReq.Token == "" {
			logger.Warn("Token field is missing in the request")
			json.WriteError(w, http.StatusBadRequest, "Invalid token field")
			return
		}

		userID := authHeader
		err := storage.AddUserToken(userID, tokenReq.Token)
		if err != nil {
			logger.Error("Failed to add token to storage", slog.String("error", err.Error()), slog.String("user_id", userID))
			json.WriteError(w, http.StatusInternalServerError, "Failed to save token")
			return
		}

		response := map[string]string{"response": "Token added successfully"}
		logger.Info("Token added successfully", slog.String("user_id", userID))
		json.WriteJSON(w, http.StatusOK, response)
	}
}
