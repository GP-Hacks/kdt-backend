package tokens

import (
	"github.com/GP-Hack/kdt2024-commons/json"
	"github.com/GP-Hack/kdt2024-gateway/internal/storage"
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
		logger := log.With(slog.String("op", op), slog.Any("request_id", reqID), slog.Any("ip", r.RemoteAddr))

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			json.WriteError(w, http.StatusUnauthorized, "Authorization required")
			return
		}

		var tokenReq TokenRequest
		if err := json.ReadJSON(r, &tokenReq); err != nil {
			logger.Error("Failed to read JSON", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if tokenReq.Token == "" {
			logger.Warn("Request to set token without token")
			json.WriteError(w, http.StatusBadRequest, "Invalid token field")
			return
		}

		userID := authHeader
		err := storage.AddUserToken(userID, tokenReq.Token)
		if err != nil {
			logger.Error("Failed to add token to MongoDB", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Failed to save token")
			return
		}

		json.WriteJSON(w, http.StatusOK, map[string]string{"response": "Token added successfully"})
		logger.Debug("Token added successfully")
	}
}
