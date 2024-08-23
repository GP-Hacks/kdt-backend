FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY chat/go.mod chat/go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/chat/cmd/chat
RUN go build -o chat_service

FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/chat/cmd/chat/chat_service .

EXPOSE 8080

CMD ["./chat_service"]
