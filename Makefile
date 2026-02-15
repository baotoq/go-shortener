.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: run-url run-analytics run-consumer gen-url gen-analytics gen-url-model gen-clicks-model run-all db-up db-down db-migrate kafka-up kafka-down

run-url: ## Run URL API service
	go run services/url-api/url.go -f services/url-api/etc/url.yaml

run-analytics: ## Run Analytics RPC service
	go run services/analytics-rpc/analytics.go -f services/analytics-rpc/etc/analytics.yaml

run-consumer: ## Run Analytics Consumer service
	go run services/analytics-consumer/consumer.go -f services/analytics-consumer/etc/consumer.yaml

gen-url: ## Regenerate URL API from .api spec
	cd services/url-api && goctl api go -api url.api -dir . -style gozero

gen-analytics: ## Regenerate Analytics RPC from .proto spec
	cd services/analytics-rpc && goctl rpc protoc analytics.proto --go_out=. --go-grpc_out=. --zrpc_out=. --style gozero

run-all: ## Run all services (URL API + Analytics RPC + Consumer)
	@echo "Starting Analytics RPC on :8081..."
	@go run services/analytics-rpc/analytics.go -f services/analytics-rpc/etc/analytics.yaml &
	@echo "Starting Analytics Consumer..."
	@go run services/analytics-consumer/consumer.go -f services/analytics-consumer/etc/consumer.yaml &
	@echo "Starting URL API on :8080..."
	@go run services/url-api/url.go -f services/url-api/etc/url.yaml

db-up: ## Start PostgreSQL
	docker compose up -d postgres

db-down: ## Stop PostgreSQL
	docker compose down

db-migrate: ## Apply database migrations
	docker compose exec -T postgres psql -U postgres -d shortener < services/migrations/000001_create_urls.up.sql
	docker compose exec -T postgres psql -U postgres -d shortener < services/migrations/000002_create_clicks.up.sql
	docker compose exec -T postgres psql -U postgres -d shortener < services/migrations/000003_add_clicks_enrichment.up.sql

kafka-up: ## Start Kafka
	docker compose up -d kafka

kafka-down: ## Stop Kafka
	docker compose down kafka

gen-url-model: ## Generate URL model from PostgreSQL (requires db-up)
	goctl model pg datasource --url "postgres://postgres:postgres@localhost:5433/shortener?sslmode=disable" --table "urls" --dir services/url-api/model --style gozero

gen-clicks-model: ## Generate clicks model from PostgreSQL (requires db-up)
	goctl model pg datasource --url "postgres://postgres:postgres@localhost:5433/shortener?sslmode=disable" --table "clicks" --dir services/analytics-rpc/model --style gozero
