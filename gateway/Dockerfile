FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY gateway/go.mod gateway/go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/gateway/cmd/gateway
RUN go build -o gateway_service

FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/gateway/cmd/gateway/gateway_service .
COPY --from=builder /app/gateway/cmd/docs/swagger.yaml .

EXPOSE 8080

CMD ["./gateway_service"]
