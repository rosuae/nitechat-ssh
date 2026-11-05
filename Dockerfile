FROM golang:1.27.1-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /nite-chat ./cmd/nite-chat

FROM alpine:3.22

RUN adduser -D -H app
WORKDIR /app
RUN mkdir -p /app/.ssh && chown -R app:app /app

COPY --from=builder /nite-chat /nite-chat

USER app
ENV HOST_KEY_PATH=/app/.ssh/host_ed25519
EXPOSE 2222

ENTRYPOINT ["/nite-chat"]