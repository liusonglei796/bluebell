FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bluebell_app ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bluebell_consumer ./cmd/consumer

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/bluebell_app /app/bluebell_app
COPY --from=builder /app/bluebell_consumer /app/bluebell_consumer
COPY config.docker.toml /app/config.toml

EXPOSE 8080

ENTRYPOINT ["/app/bluebell_app"]
CMD ["-conf", "/app/config.toml"]
