.PHONY: help install install-tools build build-all run repl dev \
	test test-integration test-coverage bench lint format vet security check \
	proto thrift generate \
	docker-build docker-up docker-down docker-logs docker-restart docker-ps docker-scale-gateway \
	k8s-apply-dev k8s-apply-staging k8s-apply-prod k8s-apply-local-kind k8s-delete k8s-status k8s-rollout-status k8s-rollback \
	kind-create kind-load kind-delete \
	tf-init tf-plan tf-validate \
	openapi-validate postman-run \
	migrate db-shell clean init version

# Variables
APP_NAME     := post-analyzer
SERVICES     := gateway postsvc authsvc analytics-consumer reanalysis-worker notification-consumer
BINARY       := $(APP_NAME)
DOCKER_IMAGE := $(APP_NAME):latest
GO           := go
GOFLAGS      := -v
LDFLAGS      := -w -s
COMPOSE      := docker compose
KUBECTL      := kubectl

# Colors for output
BLUE   := \033[0;34m
GREEN  := \033[0;32m
YELLOW := \033[0;33m
NC     := \033[0m # No Color

## help: Display this help message
help:
	@echo "$(BLUE)Post Analyzer Webserver - Makefile Commands$(NC)"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /' | column -t -s ':'

# --- Setup ------------------------------------------------------------

## install: Download and verify Go module dependencies
install:
	@echo "$(GREEN)Installing dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod verify

## install-tools: Install development tools (lint, codegen, security, docs)
install-tools:
	@echo "$(GREEN)Installing development tools...$(NC)"
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	$(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	$(GO) install github.com/cloudwego/kitex/tool/cmd/kitex@latest
	$(GO) install github.com/cloudwego/thriftgo@latest
	@echo "$(YELLOW)protoc + protoc-gen-go must be installed separately (e.g. brew install protobuf && go install google.golang.org/protobuf/cmd/protoc-gen-go@latest)$(NC)"

## init: Bootstrap a local dev environment
init: install install-tools
	@if [ ! -f .env ]; then cp .env.example .env && echo "$(GREEN)Created .env from .env.example$(NC)"; else echo "$(YELLOW).env already exists, leaving as-is$(NC)"; fi
	@echo "$(GREEN)Development environment ready! Run 'make docker-up' to start the full stack.$(NC)"

# --- Codegen (Thrift IDL / protobuf) -----------------------------------

## thrift: Regenerate Kitex/Thrift server+client stubs from idl/thrift/*.thrift into kitex_gen/
thrift:
	@echo "$(GREEN)Generating Kitex/Thrift code...$(NC)"
	cd cmd/postsvc && kitex -module $(shell head -1 go.mod | awk '{print $$2}') -service postsvc -gen-path ../../kitex_gen ../../idl/thrift/post.thrift
	cd cmd/authsvc && kitex -module $(shell head -1 go.mod | awk '{print $$2}') -service authsvc -gen-path ../../kitex_gen ../../idl/thrift/auth.thrift

## proto: Regenerate protobuf Go types from idl/proto/*.proto into internal/gen/
proto:
	@echo "$(GREEN)Generating protobuf code...$(NC)"
	protoc --go_out=. --go_opt=module=$(shell head -1 go.mod | awk '{print $$2}') idl/proto/events.proto

## generate: Regenerate all Thrift + protobuf code
generate: thrift proto

# --- Build / run --------------------------------------------------------

## build: Build the gateway binary (server + CLI + REPL entrypoint, same binary)
build:
	@echo "$(GREEN)Building $(APP_NAME)...$(NC)"
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/gateway
	@echo "$(GREEN)Build complete: $(BINARY)$(NC)"

## build-all: Build every service binary (gateway, postsvc, authsvc, analytics-consumer, reanalysis-worker, notification-consumer) into bin/
build-all:
	@echo "$(GREEN)Building all services...$(NC)"
	@mkdir -p bin
	@for svc in $(SERVICES); do \
		echo "  -> bin/$$svc"; \
		$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o bin/$$svc ./cmd/$$svc || exit 1; \
	done
	@echo "$(GREEN)Build complete: bin/{$(SERVICES)}$(NC)"

## run: Build and run the gateway in server mode (default when no CLI subcommand is given)
run: build
	@echo "$(GREEN)Running $(APP_NAME) (server mode)...$(NC)"
	./$(BINARY)

## repl: Build and drop into the interactive REPL against a running gateway (default http://localhost:8080)
repl: build
	./$(BINARY) repl

## dev: Run the gateway in development mode with file watching (falls back to plain go run)
dev:
	@echo "$(GREEN)Running in development mode...$(NC)"
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "$(YELLOW)air not installed (go install github.com/air-verse/air@latest). Running normally...$(NC)"; \
		$(GO) run ./cmd/gateway; \
	fi

# --- Testing -------------------------------------------------------------

## test: Run all unit tests with race detection + coverage
test:
	@echo "$(GREEN)Running unit tests...$(NC)"
	$(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

## test-integration: Run integration tests against a real local Postgres (needs DB_* env vars or the docker-compose defaults)
test-integration:
	@echo "$(GREEN)Running integration tests (real Postgres required)...$(NC)"
	$(GO) test -tags=integration -v ./internal/storage/...

## test-coverage: Run tests and generate an HTML coverage report
test-coverage: test
	@echo "$(GREEN)Generating coverage report...$(NC)"
	$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

## bench: Run benchmarks
bench:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	$(GO) test -bench=. -benchmem ./...

## postman-run: Run the Postman collection with newman against a running stack (override BASE_URL as needed)
BASE_URL ?= http://localhost
postman-run:
	@echo "$(GREEN)Running Postman collection against $(BASE_URL)...$(NC)"
	newman run postman/Post-Analyzer.postman_collection.json -e postman/Post-Analyzer.postman_environment.json --env-var "baseUrl=$(BASE_URL)"

## openapi-validate: Validate the OpenAPI spec
openapi-validate:
	@echo "$(GREEN)Validating OpenAPI spec...$(NC)"
	@if command -v openapi-spec-validator > /dev/null; then \
		openapi-spec-validator api-docs.yaml; \
	else \
		echo "$(YELLOW)openapi-spec-validator not installed (pip install openapi-spec-validator)$(NC)"; \
	fi

# --- Code quality ----------------------------------------------------------

## lint: Run golangci-lint
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	golangci-lint run --timeout=5m ./...

## vet: Run go vet
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	$(GO) vet ./...

## format: Format Go code (gofmt + go fmt)
format:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GO) fmt ./...
	gofmt -s -w .

## security: Run gosec static security analysis
security:
	@echo "$(GREEN)Running security checks...$(NC)"
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "$(YELLOW)gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest$(NC)"; \
	fi

## check: Run vet, lint, unit tests, and security checks — the full pre-commit gate
check: vet lint test security
	@echo "$(GREEN)All checks passed!$(NC)"

# --- Docker / Compose (full local stack: Postgres, Redis, Kafka, RabbitMQ,
#     RocketMQ, MinIO, Triton, nginx, Prometheus, Grafana, all 6 services) --

## docker-build: Build every service's Docker image via docker compose
docker-build:
	@echo "$(GREEN)Building all service images...$(NC)"
	$(COMPOSE) build

## docker-up: Start the full stack with Docker Compose
docker-up:
	@echo "$(GREEN)Starting the full stack...$(NC)"
	$(COMPOSE) up -d
	@echo "$(GREEN)Gateway (via nginx):     http://localhost/$(NC)"
	@echo "$(GREEN)Dashboard:               http://localhost/dashboard$(NC)"
	@echo "$(GREEN)Prometheus:              http://localhost:9090$(NC)"
	@echo "$(GREEN)Grafana:                 http://localhost:3000 (admin/admin)$(NC)"
	@echo "$(GREEN)MinIO console:           http://localhost:9001 (minioadmin/minioadmin)$(NC)"
	@echo "$(GREEN)RabbitMQ management:     http://localhost:15672 (guest/guest)$(NC)"

## docker-down: Stop the full stack (add ARGS=-v to also wipe volumes)
docker-down:
	@echo "$(YELLOW)Stopping services...$(NC)"
	$(COMPOSE) down $(ARGS)

## docker-logs: Tail logs from every service
docker-logs:
	$(COMPOSE) logs -f

## docker-restart: Restart the full stack
docker-restart: docker-down docker-up

## docker-ps: Show status of every container in the stack
docker-ps:
	$(COMPOSE) ps

## docker-scale-gateway: Scale the gateway service (usage: make docker-scale-gateway N=3)
N ?= 2
docker-scale-gateway:
	$(COMPOSE) up -d --scale gateway=$(N) --no-recreate

# --- Kubernetes (Kustomize base + dev/staging/prod/local-kind overlays) ----

## k8s-apply-dev: Apply the dev overlay to the current kubectl context
k8s-apply-dev:
	$(KUBECTL) apply -k deployments/k8s/overlays/dev

## k8s-apply-staging: Apply the staging overlay to the current kubectl context
k8s-apply-staging:
	$(KUBECTL) apply -k deployments/k8s/overlays/staging

## k8s-apply-prod: Apply the prod overlay (includes PDB + HPA) to the current kubectl context
k8s-apply-prod:
	$(KUBECTL) apply -k deployments/k8s/overlays/prod

## k8s-apply-local-kind: Apply the local-kind overlay (imagePullPolicy: Never, locally-built images)
k8s-apply-local-kind:
	$(KUBECTL) apply -k deployments/k8s/overlays/local-kind

## k8s-delete: Delete everything in the post-analyzer namespace (usage: make k8s-delete OVERLAY=dev)
OVERLAY ?= dev
k8s-delete:
	$(KUBECTL) delete -k deployments/k8s/overlays/$(OVERLAY)

## k8s-status: Show pods/services/ingress in the post-analyzer namespace
k8s-status:
	$(KUBECTL) get pods,svc,ingress -n post-analyzer

## k8s-rollout-status: Watch the gateway deployment rollout
k8s-rollout-status:
	$(KUBECTL) rollout status deployment/gateway -n post-analyzer

## k8s-rollback: Roll the gateway deployment back to its previous revision
k8s-rollback:
	$(KUBECTL) rollout undo deployment/gateway -n post-analyzer

## kind-create: Create a local kind cluster named post-analyzer
kind-create:
	kind create cluster --name post-analyzer

## kind-load: Build service images and load them into the kind cluster
kind-load: docker-build
	@for svc in $(SERVICES); do \
		kind load docker-image postanalyzerwebserver-$$svc:latest --name post-analyzer; \
	done

## kind-delete: Delete the local kind cluster
kind-delete:
	kind delete cluster --name post-analyzer

# --- Terraform (AWS / Azure / OCI / GCP — provisioning not applied by default) ---

## tf-init: terraform init for a given cloud (usage: make tf-init CLOUD=aws)
CLOUD ?= aws
tf-init:
	cd deployments/terraform/environments/$(CLOUD) && terraform init

## tf-plan: terraform plan for a given cloud (usage: make tf-plan CLOUD=aws)
tf-plan:
	cd deployments/terraform/environments/$(CLOUD) && terraform plan

## tf-validate: terraform validate for every cloud module (aws, azure, oci, gcp)
tf-validate:
	@for c in aws azure oci gcp; do \
		echo "$(GREEN)Validating $$c...$(NC)"; \
		(cd deployments/terraform/environments/$$c && terraform init -backend=false -input=false > /dev/null && terraform validate) || exit 1; \
	done

# --- Database --------------------------------------------------------------

## migrate: Database migrations run automatically on service startup (internal/migrations)
migrate:
	@echo "$(YELLOW)Migrations run automatically when postsvc/gateway start (see internal/migrations).$(NC)"

## db-shell: Connect to the Dockerized PostgreSQL instance
db-shell:
	$(COMPOSE) exec postgres psql -U postgres -d postanalyzer

# --- Misc --------------------------------------------------------------

## clean: Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -f $(BINARY)
	rm -rf bin/
	rm -f coverage.txt coverage.html
	rm -rf dist/
	$(GO) clean

## version: Display version information
version:
	@echo "$(BLUE)Post Analyzer Webserver$(NC)"
	@echo "Go version: $$($(GO) version)"
	@echo "Git commit: $$(git rev-parse --short HEAD 2>/dev/null || echo 'N/A')"

.DEFAULT_GOAL := help
