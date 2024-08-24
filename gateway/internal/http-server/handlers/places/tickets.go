package places

import (
	"github.com/GP-Hacks/kdt2024-commons/api/proto"
	"github.com/GP-Hacks/kdt2024-commons/json"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"net/http"
)

type Ticket struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Location  string `json:"location"`
	EventTime string `json:"event_time"`
}

func NewGetTicketsHandler(log *slog.Logger, placesClient proto.PlacesServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.places.get.New"
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		logger := log.With(
			slog.String("operation", op),
			slog.String("request_id", reqID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
		)

		logger.Info("Processing request to get places")

		select {
		case <-ctx.Done():
			logger.Warn("Request was cancelled by the client", slog.String("reason", ctx.Err().Error()))
			http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
			return
		default:
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			logger.Warn("Authorization header is missing or empty")
			json.WriteError(w, http.StatusUnauthorized, "Authorization required")
			return
		}

		request := proto.GetTicketsRequest{Token: token}

		resp, err := placesClient.GetTickets(ctx, &request)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				logger.Warn("No tickets found")
				json.WriteError(w, http.StatusNotFound, "No tickets found")
				return
			}
			logger.Error("Failed to retrieve tickets from gRPC service", slog.String("error", err.Error()))
			json.WriteError(w, http.StatusInternalServerError, "Could not retrieve tickets")
			return
		}
		var response []Ticket
		for _, ticket := range resp.GetResponse() {
			respTicket := Ticket{
				ID:        int(ticket.Id),
				Name:      ticket.Name,
				Location:  ticket.Location,
				EventTime: ticket.Timestamp.AsTime().Format("2006-01-02 15:04:05"),
			}
			response = append(response, respTicket)
		}

		logger.Debug("Places successfully retrieved", slog.Any("response", response))
		json.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"response": response,
		})
	}
}
