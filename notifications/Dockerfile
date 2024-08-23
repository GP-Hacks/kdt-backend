FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY notifications/go.mod notifications/go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/notifications/cmd/notifications
RUN go build -o notifications_service

FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/notifications/cmd/notifications/notifications_service .

EXPOSE 8080

CMD ["./notifications_service"]
