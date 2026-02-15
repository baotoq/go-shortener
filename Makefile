.PHONY: run-url run-analytics run-all build generate-mocks docker-up docker-down

# Run URL Service with Dapr sidecar
run-url:
	dapr run --app-id url-service --app-port 8080 --dapr-http-port 3500 --dapr-grpc-port 50001 --resources-path ./dapr/components -- go run ./cmd/url-service

# Run Analytics Service with Dapr sidecar
run-analytics:
	dapr run --app-id analytics-service --app-port 8081 --dapr-http-port 3501 --dapr-grpc-port 50002 --resources-path ./dapr/components -- go run ./cmd/analytics-service

# Run both services using Dapr multi-app run
run-all:
	dapr run -f dapr.yaml

# Build both binaries
build:
	go build -o bin/url-service ./cmd/url-service
	go build -o bin/analytics-service ./cmd/analytics-service

# Generate mocks for all interfaces
generate-mocks:
	mockery --config .mockery.yaml

# Start all services with Docker Compose
docker-up:
	docker compose up --build -d

# Stop all services
docker-down:
	docker compose down
