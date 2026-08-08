# Post Analyzer Webserver

<p align="center">
   <img src="assets/post-ui.png" alt="Post Analyzer" width="100%" style="border-radius: 8px;">
</p>

<p align="center">
   <img src="https://img.shields.io/badge/License-MIT-green" alt="License">
   <img src="https://img.shields.io/badge/Go-1.25-blue?style=flat&logo=go" alt="Go version">
   <img src="https://img.shields.io/badge/Status-Production%20Ready-success" alt="Status">
   <img src="https://img.shields.io/badge/Version-3.0.0-blue" alt="Version">
   <img src="https://img.shields.io/badge/Architecture-Microservices-orange" alt="Architecture">
</p>

An enterprise-grade, microservices-based backend for storing, serving, and analyzing "posts" (title/body records with per-character-frequency analytics and ML-driven sentiment classification). What started as a single-file Go web server is now a full CloudWeGo-based (Kitex + Hertz + Sonic + Netpoll) microservices system with RPC and HTTP surfaces, ABAC authorization, three distinct message brokers, local S3-compatible object storage, an Nvidia Triton ML integration, full observability, and Kubernetes/Terraform deployment artifacts for AWS, Azure, and OCI.

It's still fundamentally about **post analysis** — every piece of infrastructure here (Kafka event stream, RabbitMQ reanalysis queue, RocketMQ scheduled rechecks, Redis cache, Triton sentiment model) exists in direct service of that core CRUD-and-analyze use case, not as unrelated scaffolding.

## Table of contents

- [Architecture](#architecture)
- [Tech stack](#tech-stack)
- [Messaging: why three brokers](#messaging-why-three-brokers)
- [Quickstart](#quickstart)
- [Interacting with it: HTTP, RPC, CLI, REPL, UI](#interacting-with-it-http-rpc-cli-repl-ui)
- [Project structure](#project-structure)
- [Configuration](#configuration)
- [Testing](#testing)
- [Deployment](#deployment)
- [API documentation](#api-documentation)
- [Makefile reference](#makefile-reference)
- [License](#license)

## Architecture

```
                              ┌─────────────┐
                              │   Browser   │
                              │ (UI + Dash) │
                              └──────┬──────┘
                                     │ HTTP
                     ┌───────────────┴────────────────┐
                     │        nginx (edge LB)          │  ← scales gateway via
                     │  resolver-based upstream, so     │    docker compose --scale
                     │  `--scale gateway=N` load-        │
                     │  balances automatically           │
                     └───────────────┬────────────────┘
                                     │
                     ┌───────────────┴────────────────┐
                     │     gateway (Hertz / Netpoll)    │  server / CLI / REPL —
                     │  REST API · ABAC middleware ·    │  same binary, 3 modes
                     │  rate limit · panic recovery ·   │
                     │  metrics · CORS · compression    │
                     └──────┬────────────────────┬─────┘
                             │ Kitex RPC (Thrift,          │ Kitex RPC
                             │ TTHeader, mux)               │
                  ┌──────────┴──────────┐      ┌───────────┴───────────┐
                  │       postsvc        │      │        authsvc         │
                  │ CRUD · analysis ·    │      │ JWT issuance · ABAC     │
                  │ Redis cache ·        │      │ policy decisions        │
                  │ Kafka + RocketMQ     │      └────────────────────────┘
                  │ producers · Triton   │
                  └───┬────┬────┬───┬───┘
                      │    │    │   │
        ┌─────────────┘    │    │   └──────────────┐
        │                  │    │                   │
   ┌────▼────┐      ┌──────▼┐  ┌▼───────┐    ┌──────▼──────┐
   │ Postgres │      │ Redis │  │ Kafka  │    │  RocketMQ    │
   │ (or file)│      │ cache │  │ events │    │  (delayed)   │
   └──────────┘      └───────┘  └───┬────┘    └──────┬───────┘
                                    │                 │
                          ┌─────────▼────────┐ ┌──────▼──────────────┐
                          │ analytics-consumer│ │ notification-consumer│
                          └────────────────────┘ └──────────────────────┘

   RabbitMQ (reanalysis.jobs) ⇄ reanalysis-worker ──calls──▶ postsvc
   MinIO (S3-compatible)      ⇄ postsvc (export persistence)
   Triton (ONNX sentiment)    ⇄ postsvc (enrichment on post creation)
   Prometheus ⇄ every service's /metrics · Grafana ⇄ Prometheus datasource
```

Every service is independently deployable (its own Dockerfile stage, its own Kubernetes Deployment, its own `/metrics` and health endpoints) but all six binaries live in this one repo/module, built via the same shared `Dockerfile` (`ARG SERVICE=...`).

### CDN / edge readiness

Not part of the local stack (no cloud credentials are used by this repo), but the Terraform is structured for it: the AWS module includes an optional CloudFront distribution (`enable_cdn`) and the Azure module an optional Front Door profile (`enable_cdn`), both off by default since their origin is the ingress LoadBalancer hostname that only exists after a real cluster is stood up and `deployments/k8s/` is applied to it — a genuine two-phase deploy, not something fakeable ahead of time. Both cache static UI assets at the edge while explicitly excluding `/api/*` from caching, since those responses are per-user and ABAC-gated. OCI has no equivalent first-party resource in its Terraform provider; see the comment in `deployments/terraform/modules/oci/main.tf` for the recommended path (a third-party CDN in front of the OCI Load Balancer).

## Tech stack

| Concern | Technology |
|---|---|
| RPC framework | [Kitex](https://www.cloudwego.io/docs/kitex/) — Thrift IDL, TTHeader transport, connection multiplexing |
| HTTP framework | [Hertz](https://www.cloudwego.io/docs/hertz/) — built on Netpoll |
| JSON | [Sonic](https://github.com/bytedance/sonic) (via Hertz/Kitex) |
| IDL | Thrift (`idl/thrift/*.thrift`) for RPC contracts, Protobuf (`idl/proto/events.proto`) for message-bus payloads |
| Auth | JWT (golang-jwt/v5) + custom ABAC policy-decision-point (`internal/abac`), default-deny |
| Database | PostgreSQL (works against any local Postgres — see [Configuration](#configuration)) or flat-file JSON store |
| Cache | Redis, with automatic in-memory fallback if unreachable |
| Messaging | Kafka, RabbitMQ, RocketMQ — see [why three brokers](#messaging-why-three-brokers) |
| Object storage | MinIO (local, S3-API-compatible — no cloud storage dependency) |
| ML | Nvidia Triton Inference Server (CPU), serving a trained TF-IDF + LogisticRegression ONNX sentiment classifier |
| Edge / LB | nginx (resolver-based upstream so Compose scaling load-balances) |
| Observability | Prometheus + Grafana (provisioned dashboard), structured JSON logging, request IDs |
| API surface | REST (OpenAPI 3.0.3, `api-docs.yaml`) + Kitex RPC + CLI + REPL |
| Containers | Docker, Docker Compose (16-container full stack) |
| Orchestration | Kubernetes — Kustomize base + dev/staging/prod/local-kind overlays, Ingress, Argo Rollouts (canary/blue-green), Istio mesh manifests |
| IaC | Terraform modules for AWS (EKS/RDS/ElastiCache/ECR/CloudFront), Azure (AKS/Postgres Flexible/Redis Cache/ACR/Front Door), OCI (OKE/PSQL/OCIR) |
| CI | GitHub Actions — lint, unit + integration tests, per-service build matrix, per-service Docker image matrix, Trivy security scan |

## Messaging: why three brokers

Each broker was picked for a distinct pattern it's actually good at — not "one topic per broker" for its own sake:

- **Kafka** (`post.events`) — a durable, replayable, multi-consumer **event log**. `postsvc` publishes on every create/update/delete/analyze; `analytics-consumer` is one subscriber today, but the topic is designed so other consumers (a warehouse loader, a search indexer) could subscribe independently without `postsvc` knowing or caring.
- **RabbitMQ** (`reanalysis.jobs`) — a classic **work queue**. `POST /api/v1/posts/reanalyze` enqueues a job; `reanalysis-worker` is a competing consumer that acks/nacks-with-requeue. This is a task to be done exactly by one worker, not an event for many subscribers — the wrong fit for Kafka.
- **RocketMQ** (`post-notifications`) — **native delayed/scheduled delivery**. On post creation, a "recheck this post" notification is scheduled ~10s out using RocketMQ's delay-level messages, something Kafka and RabbitMQ don't support natively without extra machinery (a DLX-based delay queue, a scheduler polling a table, etc.).

## Quickstart

Requires Docker + Docker Compose. Nothing else needs to be pre-installed to run the full stack.

```bash
git clone https://github.com/hoangsonww/Post-Analyzer-Webserver.git
cd Post-Analyzer-Webserver

make docker-up      # or: docker compose up -d
```

This starts all 17 containers: Postgres, Redis, Kafka, RabbitMQ, RocketMQ (nameserver + broker), MinIO, Nvidia Triton, nginx, Prometheus, Grafana, and all 6 app services (gateway, postsvc, authsvc, analytics-consumer, reanalysis-worker, notification-consumer). First run pulls the Triton image (~13GB) and can take a few minutes.

Once healthy:

| Surface | URL |
|---|---|
| Web UI | http://localhost/ |
| Analytics dashboard | http://localhost/dashboard |
| REST API | http://localhost/api/v1/ |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 (`admin`/`admin`) |
| MinIO console | http://localhost:9001 (`minioadmin`/`minioadmin`) |
| RabbitMQ management | http://localhost:15672 (`guest`/`guest`) |

Demo accounts (seeded on `authsvc` startup): `admin`/`admin123`, `editor`/`editor123`, `viewer`/`viewer123` — three distinct ABAC roles, see `internal/abac/policy.go`.

### Running against your own local Postgres instead of Compose's

The default `.env`/Compose config uses its own Postgres container, but nothing here is hardcoded to that — point `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME` (see `.env.example`) at any Postgres instance you already have running locally and `make run` (or `go run ./cmd/gateway`) will create its schema there on first connect via `internal/migrations`.

## Interacting with it: HTTP, RPC, CLI, REPL, UI

The gateway binary supports all of these from the same executable:

```bash
go build -o post-analyzer ./cmd/gateway

./post-analyzer                       # server mode (default) — HTTP + drives RPC to postsvc/authsvc
./post-analyzer login admin admin123  # one-shot CLI
./post-analyzer posts list
./post-analyzer posts create --title "Hello" --body "World" --user-id 1
./post-analyzer sentiment "this is great"
./post-analyzer repl                  # interactive REPL — same command tree, one process
```

RPC (Kitex/Thrift/TTHeader) is what the gateway itself speaks to `postsvc`/`authsvc` internally — see `internal/rpcclient` for the plain-Go interfaces that hide this from the rest of the gateway code.

## Project structure

```
cmd/                    6 service entrypoints (gateway, postsvc, authsvc, analytics-consumer, reanalysis-worker, notification-consumer)
internal/
  abac/                 ABAC policy engine, JWT issuance, demo user store
  adapt/                Thrift <-> domain model conversion
  api/                  REST handlers + router
  bootstrap/            shared storage/migration bootstrap
  cache/                Redis + in-memory cache implementations
  cli/                  CLI/REPL command tree (cobra), also hosts `serve.go` (the HTTP server)
  export/                JSON/CSV export writers
  handlers/              legacy web UI handlers (home.html)
  messaging/             kafka/, rabbitmq/, rocketmq/ client wrappers
  metrics/               Prometheus collectors + HTTP metrics middleware
  middleware/            ABAC, rate limit, timeout, recovery, logging, CORS, compression, security headers
  ml/triton/              Triton KServe v2 HTTP client
  objectstore/            MinIO client wrapper
  rpcclient/              Kitex client wrappers exposing plain Go interfaces
  service/                PostService business logic (filter/sort/paginate/analyze)
  storage/                Postgres + file storage backends
idl/
  thrift/                 PostService + AuthService Thrift IDL
  proto/                  Kafka/RocketMQ message payloads (protobuf)
kitex_gen/                generated Kitex/Thrift code (committed)
deployments/
  k8s/                    Kustomize base + dev/staging/prod/local-kind overlays, Argo Rollouts, Istio mesh
  terraform/               AWS / Azure / OCI modules + environments
  nginx/, grafana/, triton/, rocketmq/
assets/                   dashboard.html, home.html, static UI assets
postman/                   Postman collection + environment
api-docs.yaml               OpenAPI 3.0.3 spec
docker-compose.yml           full local stack (16 containers)
Dockerfile                    single parameterized multi-stage build (ARG SERVICE=...)
Makefile
```

## Configuration

All configuration is environment-variable driven — see `.env.example` for the full, current list (server, database, security, logging, RPC addresses, JWT/ABAC, Redis, Kafka/RabbitMQ/RocketMQ, MinIO, Triton). Every optional integration (Redis, each message broker, MinIO, Triton) degrades gracefully when disabled or unreachable rather than failing startup — the service logs a warning and the dependent feature returns a clear error/503 instead.

## Testing

```bash
make test               # unit tests (race detector + coverage), no external dependencies
make test-integration   # storage integration tests against a real local Postgres
make postman-run        # Postman/newman collection against a running stack (BASE_URL=http://localhost by default)
make lint                # golangci-lint (0 issues)
make check                # vet + lint + test + gosec — full pre-commit gate
```

## Deployment

- **Docker Compose** — `make docker-up` / `docker compose up -d`, the full local stack.
- **Kubernetes** — `deployments/k8s/base` (Kustomize) with `overlays/{dev,staging,prod,local-kind}`. `make k8s-apply-dev` etc. Includes an Ingress, a `prod` overlay with PodDisruptionBudget + HorizontalPodAutoscaler, Argo Rollouts manifests for canary/blue-green (`deployments/k8s/rollout/`), and Istio mesh manifests (`deployments/k8s/mesh/`).
- **Local kind cluster** — `make kind-create && make kind-load && make k8s-apply-local-kind`.
- **Terraform (AWS / Azure / OCI)** — `deployments/terraform/environments/{aws,azure,oci}`, each provisioning the managed equivalent of the local stack (managed Kubernetes, managed Postgres, managed Redis, container registry, optional CDN). `make tf-validate` validates all three against their real provider schemas. Not applied by default — no cloud credentials are used by this repo; see each environment's `*.tfvars.example`.

## API documentation

- **OpenAPI 3.0.3**: `api-docs.yaml` (validated with `openapi-spec-validator`)
- **Postman collection**: `postman/Post-Analyzer.postman_collection.json` + `postman/Post-Analyzer.postman_environment.json` — 27 requests covering CRUD, bulk ops, export, analytics, ABAC role checks (viewer/editor/admin), ML sentiment, and object-storage export listing/download. Runs clean via `make postman-run`.

## Makefile reference

Run `make help` for the full, current list (build, codegen, test, lint, Docker, Kubernetes, kind, Terraform, Postman/OpenAPI validation targets).

## License

Distributed under the MIT License. See `LICENSE` for details.

---

Created by [Son Nguyen](https://github.com/hoangsonww).
