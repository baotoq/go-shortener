FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /app/go-shortener ./cmd/go-shortener

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/go-shortener /app/go-shortener
COPY configs/config.yaml /data/conf/config.yaml

WORKDIR /app

EXPOSE 8000
EXPOSE 9000

CMD ["./go-shortener", "-conf", "/data/conf/config.yaml"]
