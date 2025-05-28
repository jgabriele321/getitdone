FROM golang:1.24.2-alpine AS builder
WORKDIR /app
COPY go-todo-bot/go.mod go-todo-bot/go.sum ./
RUN go mod download
COPY go-todo-bot/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/bot

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=UTC
ENV PORT=8080
RUN adduser -D -s /bin/sh appuser
WORKDIR /app
COPY --from=builder /app/main .
COPY go-todo-bot/env.example .env.example
RUN chown appuser:appuser /app/main
USER appuser
EXPOSE ${PORT}
CMD ["./main"]