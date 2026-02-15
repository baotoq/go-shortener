.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: run-url run-analytics gen-url gen-analytics run-all

run-url: ## Run URL API service
	go run services/url-api/url.go -f services/url-api/etc/url.yaml

run-analytics: ## Run Analytics RPC service
	go run services/analytics-rpc/analytics.go -f services/analytics-rpc/etc/analytics.yaml

gen-url: ## Regenerate URL API from .api spec
	cd services/url-api && goctl api go -api url.api -dir . -style gozero

gen-analytics: ## Regenerate Analytics RPC from .proto spec
	cd services/analytics-rpc && goctl rpc protoc analytics.proto --go_out=. --go-grpc_out=. --zrpc_out=. --style gozero

run-all: ## Run both services (URL API + Analytics RPC)
	@echo "Starting Analytics RPC on :8081..."
	@go run services/analytics-rpc/analytics.go -f services/analytics-rpc/etc/analytics.yaml &
	@echo "Starting URL API on :8080..."
	@go run services/url-api/url.go -f services/url-api/etc/url.yaml
