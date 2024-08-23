FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY votes/go.mod ./
RUN go mod download

COPY . .

WORKDIR /app/votes/cmd/votes
RUN go build -o votes_service

FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/votes/cmd/votes/votes_service .

EXPOSE 8080

CMD ["./votes_service"]
