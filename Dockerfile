# ========== Build stage ==========
FROM golang:1.24-alpine AS builder
WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build all services
COPY . .

# Build each service as a separate binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/url-api ./services/url-api/url.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/analytics-rpc ./services/analytics-rpc/analytics.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/analytics-consumer ./services/analytics-consumer/consumer.go

# ========== URL API ==========
FROM alpine:3.20 AS url-api
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/url-api /bin/url-api
COPY services/url-api/etc /etc/url-api
EXPOSE 8080 6470
CMD ["/bin/url-api", "-f", "/etc/url-api/url-docker.yaml"]

# ========== Analytics RPC ==========
FROM alpine:3.20 AS analytics-rpc
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/analytics-rpc /bin/analytics-rpc
COPY services/analytics-rpc/etc /etc/analytics-rpc
EXPOSE 8081 6471
CMD ["/bin/analytics-rpc", "-f", "/etc/analytics-rpc/analytics-docker.yaml"]

# ========== Analytics Consumer ==========
FROM alpine:3.20 AS analytics-consumer
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/analytics-consumer /bin/analytics-consumer
COPY services/analytics-consumer/etc /etc/analytics-consumer
EXPOSE 6472 8082
CMD ["/bin/analytics-consumer", "-f", "/etc/analytics-consumer/consumer-docker.yaml"]
