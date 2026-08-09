# Post Analyzer Webserver

![License](https://img.shields.io/badge/License-MIT-green) ![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go&logoColor=white) ![Status](https://img.shields.io/badge/Status-Production%20Ready-success) ![Version](https://img.shields.io/badge/Version-3.0.0-blue) ![Architecture](https://img.shields.io/badge/Architecture-Microservices-orange) ![Kitex](https://img.shields.io/badge/Kitex-RPC-1E90FF?style=flat) ![Hertz](https://img.shields.io/badge/Hertz-HTTP-1E90FF?style=flat) ![Sonic](https://img.shields.io/badge/Sonic-JSON-1E90FF?style=flat) ![Netpoll](https://img.shields.io/badge/Netpoll-Networking-1E90FF?style=flat) ![TTHeader](https://img.shields.io/badge/TTHeader-Transport-1E90FF?style=flat) ![Thrift](https://img.shields.io/badge/Thrift-IDL-6A5ACD?style=flat) ![Protobuf](https://img.shields.io/badge/Protocol%20Buffers-IDL-6A5ACD?style=flat&logo=googleprotocolbuffers&logoColor=white) ![OpenAPI](https://img.shields.io/badge/OpenAPI-3.0.3-6BA539?style=flat&logo=openapiinitiative&logoColor=white) ![Postman](https://img.shields.io/badge/Postman-Collection-FF6C37?style=flat&logo=postman&logoColor=white) ![Cobra](https://img.shields.io/badge/Cobra-CLI%2FREPL-00ADD8?style=flat&logo=go&logoColor=white) ![JWT](https://img.shields.io/badge/JWT-Auth-000000?style=flat&logo=jsonwebtokens&logoColor=white) ![ABAC](https://img.shields.io/badge/ABAC-Authorization-red?style=flat) ![bcrypt](https://img.shields.io/badge/bcrypt-Password%20Hashing-lightgrey?style=flat) ![Trivy](https://img.shields.io/badge/Trivy-Security%20Scan-1904DA?style=flat) ![PostgreSQL](https://img.shields.io/badge/PostgreSQL-Database-4169E1?style=flat&logo=postgresql&logoColor=white) ![Redis](https://img.shields.io/badge/Redis-Cache-DC382D?style=flat&logo=redis&logoColor=white) ![Kafka](https://img.shields.io/badge/Apache%20Kafka-Event%20Log-231F20?style=flat&logo=apachekafka&logoColor=white) ![RabbitMQ](https://img.shields.io/badge/RabbitMQ-Work%20Queue-FF6600?style=flat&logo=rabbitmq&logoColor=white) ![RocketMQ](https://img.shields.io/badge/RocketMQ-Delayed%20Delivery-D77310?style=flat&logo=apacherocketmq&logoColor=white) ![MinIO](https://img.shields.io/badge/MinIO-Object%20Storage-C72E49?style=flat&logo=minio&logoColor=white) ![Nvidia Triton](https://img.shields.io/badge/Nvidia%20Triton-Inference%20Server-76B900?style=flat&logo=nvidia&logoColor=white) ![ONNX](https://img.shields.io/badge/ONNX-Model%20Format-005CED?style=flat&logo=onnx&logoColor=white) ![scikit-learn](https://img.shields.io/badge/scikit--learn-Model%20Training-F7931E?style=flat&logo=scikitlearn&logoColor=white) ![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-E6522C?style=flat&logo=prometheus&logoColor=white) ![Grafana](https://img.shields.io/badge/Grafana-Dashboards-F46800?style=flat&logo=grafana&logoColor=white) ![nginx](https://img.shields.io/badge/nginx-Load%20Balancer-009639?style=flat&logo=nginx&logoColor=white) ![Docker](https://img.shields.io/badge/Docker-Containers-2496ED?style=flat&logo=docker&logoColor=white) ![Docker Compose](https://img.shields.io/badge/Docker%20Compose-17%20services-2496ED?style=flat&logo=docker&logoColor=white) ![Kubernetes](https://img.shields.io/badge/Kubernetes-Kustomize-326CE5?style=flat&logo=kubernetes&logoColor=white) ![Argo Rollouts](https://img.shields.io/badge/Argo%20Rollouts-Canary%2FBlue--Green-EF7B4D?style=flat&logo=argo&logoColor=white) ![Istio](https://img.shields.io/badge/Istio-Service%20Mesh-466BB0?style=flat&logo=istio&logoColor=white) ![Terraform](https://img.shields.io/badge/Terraform-AWS%20%7C%20Azure%20%7C%20OCI-7B42BC?style=flat&logo=terraform&logoColor=white) ![AWS](https://img.shields.io/badge/AWS-EKS%20%7C%20RDS%20%7C%20CloudFront-232F3E?style=flat&logo=amazonwebservices&logoColor=white) ![Azure](https://img.shields.io/badge/Azure-AKS%20%7C%20Front%20Door-0078D4?style=flat&logo=microsoftazure&logoColor=white) ![OCI](https://img.shields.io/badge/OCI-OKE-F80000?style=flat&logo=oracle&logoColor=white) ![GitHub Actions](https://img.shields.io/badge/GitHub%20Actions-CI-2088FF?style=flat&logo=githubactions&logoColor=white) ![golangci-lint](https://img.shields.io/badge/golangci--lint-0%20issues-brightgreen?style=flat) ![Make](https://img.shields.io/badge/Make-Build%20Automation-lightgrey?style=flat&logo=gnu&logoColor=white)

An enterprise-grade, microservices-based backend for storing, serving, and analyzing "posts" (title/body records with per-character-frequency analytics and ML-driven sentiment classification). Built on a CloudWeGo stack (Kitex + Hertz + Sonic + Netpoll) with RPC and HTTP surfaces, ABAC authorization, three distinct message brokers, local S3-compatible object storage, an Nvidia Triton ML integration, full observability, and Kubernetes/Terraform deployment artifacts for AWS, Azure, and OCI.

It's fundamentally about **post analysis** — every piece of infrastructure here (Kafka event stream, RabbitMQ reanalysis queue, RocketMQ scheduled rechecks, Redis cache, Triton sentiment model) exists in direct service of that core CRUD-and-analyze use case, not as unrelated scaffolding.

## Table of contents

- [Screenshots](#screenshots)
- [Architecture](#architecture)
- [Tech stack](#tech-stack)
- [Messaging: why three brokers](#messaging-why-three-brokers)
- [Quickstart](#quickstart)
- [Interacting with it: HTTP, RPC, CLI, REPL, UI](#interacting-with-it-http-rpc-cli-repl-ui)
- [API conventions: errors, envelopes, status codes](#api-conventions-errors-envelopes-status-codes)
- [Project structure](#project-structure)
- [Configuration reference](#configuration-reference)
- [Testing](#testing)
- [Deployment](#deployment)
- [API documentation](#api-documentation)
- [Makefile reference](#makefile-reference)
- [License](#license)

## Screenshots

Everything below is the actual running app — captured at viewport size against the real docker-compose stack, not mockups. More detail (including the ABAC role model behind the dashboard's admin panel) is in [ARCHITECTURE.md](./ARCHITECTURE.md).

| | |
|---|---|
| **Home — search, sort, paginate** ![Home](assets/ui-home.png) | **Home — dark mode** ![Home dark](assets/ui-home-dark.png) |
| **New post** ![New post modal](assets/ui-new-post-modal.png) | **Edit post** ![Edit post modal](assets/ui-edit-post-modal.png) |
| **Quick analysis (current page)** ![Quick analysis](assets/ui-quick-analysis.png) | **Full analysis (entire dataset, server-computed)** ![Full analysis](assets/ui-full-analysis.png) |
| **Dashboard — sign in** ![Dashboard login](assets/ui-dashboard-login.png) | **Dashboard — analytics** ![Dashboard overview](assets/ui-dashboard-overview.png) |
| **Dashboard — admin panel** (role-gated) ![Dashboard admin](assets/ui-dashboard-admin.png) | **No-JS fallback form** ![Add post legacy form](assets/ui-add-post-legacy.png) |

## Architecture

```mermaid
flowchart TB
    Browser(["Browser<br/>UI + Dashboard"])
    Nginx["nginx (edge LB)<br/>resolver-based upstream —<br/>scales with --scale gateway=N"]
    Gateway["gateway (Hertz / Netpoll)<br/>server · CLI · REPL — same binary, 3 modes<br/>REST API · ABAC middleware · rate limit ·<br/>panic recovery · metrics · CORS · compression"]
    PostSvc["postsvc<br/>CRUD · analysis ·<br/>Redis cache ·<br/>Kafka + RocketMQ producers · Triton"]
    AuthSvc["authsvc<br/>JWT issuance ·<br/>ABAC policy decisions"]
    PG[("Postgres<br/>(or file)")]
    Redis[("Redis cache")]
    Kafka[("Kafka<br/>events")]
    RocketMQ[("RocketMQ<br/>delayed")]
    Analytics["analytics-consumer"]
    Notification["notification-consumer"]
    RabbitMQ[("RabbitMQ<br/>reanalysis.jobs")]
    ReanalysisWorker["reanalysis-worker"]
    MinIO[("MinIO<br/>S3-compatible")]
    Triton["Nvidia Triton<br/>ONNX sentiment"]
    Prometheus["Prometheus"]
    Grafana["Grafana"]

    Browser -->|HTTP| Nginx --> Gateway
    Gateway -->|Kitex RPC, Thrift, TTHeader, mux| PostSvc
    Gateway -->|Kitex RPC| AuthSvc
    PostSvc --> PG
    PostSvc --> Redis
    PostSvc --> Kafka --> Analytics
    PostSvc --> RocketMQ --> Notification
    Gateway -->|enqueue| RabbitMQ --> ReanalysisWorker -->|Kitex RPC| PostSvc
    PostSvc <--> MinIO
    PostSvc <--> Triton
    Prometheus -.->|scrapes /metrics| Gateway
    Prometheus -.-> PostSvc
    Prometheus -.-> AuthSvc
    Grafana --> Prometheus
```

Diagrams for the pieces that are easier to follow as a picture — the full component graph, the request lifecycle through ABAC/RPC, the post-creation event fan-out, the error-propagation path from a typed service error to an HTTP response, the ABAC decision flow, and how the same six service images map onto Docker Compose vs. Kubernetes vs. Terraform — are all in **[ARCHITECTURE.md](./ARCHITECTURE.md)**.

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
| Database | PostgreSQL (works against any local Postgres — see [Configuration](#configuration-reference)) or flat-file JSON store |
| Cache | Redis, with automatic in-memory fallback if unreachable |
| Messaging | Kafka, RabbitMQ, RocketMQ — see [why three brokers](#messaging-why-three-brokers) |
| Object storage | MinIO (local, S3-API-compatible — no cloud storage dependency) |
| ML | Nvidia Triton Inference Server (CPU), serving a trained TF-IDF + LogisticRegression ONNX sentiment classifier |
| Edge / LB | nginx (resolver-based upstream so Compose scaling load-balances) |
| Observability | Prometheus + Grafana (provisioned dashboard), structured JSON logging, request IDs |
| API surface | REST (OpenAPI 3.0.3, `api-docs.yaml`) + Kitex RPC + CLI + REPL + a small JSON surface backing the built-in web UI |
| Containers | Docker, Docker Compose (17-container full stack) |
| Orchestration | Kubernetes — Kustomize base + dev/staging/prod/local-kind overlays, Ingress, Argo Rollouts (canary/blue-green), Istio mesh manifests |
| IaC | Terraform modules for AWS (EKS/RDS/ElastiCache/ECR/CloudFront), Azure (AKS/Postgres Flexible/Redis Cache/ACR/Front Door), OCI (OKE/PSQL/OCIR) |
| CI | GitHub Actions — lint, unit + integration tests, per-service build matrix, per-service Docker image matrix, Trivy security scan |

## Messaging: why three brokers

Each broker was picked for a distinct pattern it's actually good at — not "one topic per broker" for its own sake:

- **Kafka** (`post.events`) — a durable, replayable, multi-consumer **event log**. `postsvc` publishes on every create/update/delete/analyze; `analytics-consumer` is one subscriber today, but the topic is designed so other consumers (a warehouse loader, a search indexer) could subscribe independently without `postsvc` knowing or caring.
- **RabbitMQ** (`reanalysis.jobs`) — a classic **work queue**. `POST /api/v1/posts/reanalyze` enqueues a job; `reanalysis-worker` is a competing consumer that acks/nacks-with-requeue. This is a task to be done exactly by one worker, not an event for many subscribers — the wrong fit for Kafka.
- **RocketMQ** (`post-notifications`) — **native delayed/scheduled delivery**. On post creation, a "recheck this post" notification is scheduled ~10s out using RocketMQ's delay-level messages, something Kafka and RabbitMQ don't support natively without extra machinery (a DLX-based delay queue, a scheduler polling a table, etc.).

See [ARCHITECTURE.md](./ARCHITECTURE.md#post-creation-fan-out-across-three-brokers) for the full fan-out sequence diagram.

## Quickstart

Requires Docker + Docker Compose. Nothing else needs to be pre-installed to run the full stack.

```bash
git clone https://github.com/hoangsonww/Post-Analyzer.git
cd Post-Analyzer

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

Demo accounts (seeded on `authsvc` startup), three distinct ABAC roles — see `internal/abac/policy.go`:

| Username | Password | Role | Can |
|---|---|---|---|
| `admin` | `admin123` | admin | Everything, including `/api/v1/admin/status` |
| `editor` | `editor123` | editor | Create/update posts; delete requires `X-MFA-Verified: true` |
| `viewer` | `viewer123` | viewer | Read-only |

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

### CLI/REPL output is colorized

Success (✓ green), errors (✗ red), table headers (bold), IDs (cyan), timestamps (dim), and sentiment labels/bars (green/yellow/red by class) — colored automatically when stdout is a real terminal, and automatically **off** when it isn't (piped to a file, redirected in a script, captured by `| tee`, running in most CI log viewers) or when [`NO_COLOR`](https://no-color.org) is set. `--no-color` forces it off explicitly. The REPL also opens with a small banner:

```
  _____   __          __
 |  __ \ /\ \        / /
 | |__) /  \ \  /\  / /
 |  ___/ /\ \ \/  \/ /
 | |  / ____ \  /\  /
 |_| /_/    \_\/  \/

connected to http://localhost:8080
Type "help" for commands, "exit" to quit.
post-analyzer>
```

### The web UI (`/`) has real CRUD now, not just a viewer

`home.html` is a small SPA-ish page (vanilla JS, no framework, no CDN dependencies — even the character-frequency bar charts are hand-rolled, not Chart.js) backed by a set of JSON endpoints under `/web/*`:

| Method | Path | Does |
|---|---|---|
| `GET` | `/web/posts` | List, with `?search=&sortBy=&sortOrder=&page=&pageSize=` — all pushed down to postsvc, not filtered client-side |
| `GET` | `/web/posts/{id}` | Fetch one (prefills the edit modal) |
| `POST` | `/web/posts` | Create |
| `PUT` | `/web/posts/{id}` | Update |
| `DELETE` | `/web/posts/{id}` | Delete |

These are deliberately **separate from the ABAC-gated `/api/v1` surface** — same trust level as the pre-existing `/add` form post, a convenience surface for the bundled UI rather than a public API contract. External integrations should use `/api/v1` (below), which requires a JWT and is subject to ABAC.

The dashboard (`/dashboard`) is a second, independent page: sign in with any of the three demo roles to see live analytics (character frequency, posts per user, posts by time of day, recent posts), with an additional system-status panel that only renders for `admin` (see the [screenshots](#screenshots)).

## API conventions: errors, envelopes, status codes

Every JSON response — success or error, `/api/v1/*` or `/web/*` — uses the same envelope shape:

```json
{"data": {...}, "meta": {"requestId": "...", "timestamp": "..."}}
{"error": {"code": "NOT_FOUND", "message": "Post not found"}, "meta": {...}}
```

Error messages are **specific and safe to show a user**, not a blanket "something went wrong" — `code` is a stable machine-readable string (`NOT_FOUND`, `VALIDATION_FAILED`, `UNAUTHORIZED`, `FORBIDDEN`, `CONFLICT`, `SERVICE_UNAVAILABLE`, `INTERNAL_ERROR`) and `message` is meant to be displayed as-is. Concretely:

- Wrong or unregistered login credentials → `401` `"invalid username or password"` — not a generic "authentication required."
- `GET`/`PUT`/`DELETE` on a post ID that doesn't exist → `404` `"Post not found"` — not a `500`.
- A missing/invalid JWT → `401` `"missing bearer token"` or `"invalid or expired token"`.
- An ABAC denial → `403` with the **actual policy-evaluation reason** (e.g. `"role 'viewer' has no write permission on resource 'post'"`), not just "forbidden."
- Anything genuinely unexpected (a database or transport failure) → `500` `"Internal server error"`, deliberately generic — the real cause is logged server-side, never echoed to the client.

This is enforced end-to-end, including across the Kitex RPC boundary: `postsvc` and `authsvc` classify their own errors with `*errors.AppError` (a Go type carrying an HTTP status, a machine code, and a safe message), `internal/adapt.Err` encodes that onto the Thrift `BaseResp` (status in `StatusCode`, message in `StatusMessage`, code in `Extra["code"]`), and `internal/rpcclient` decodes it back into the exact same typed error on the gateway side — see [ARCHITECTURE.md](./ARCHITECTURE.md#error-propagation-a-typed-error-crossing-the-rpc-boundary) for the full sequence diagram. A raw, unclassified error (a driver/network failure) is never echoed across that boundary — only logged.

## Project structure

```mermaid
flowchart LR
    subgraph cmd["cmd/ — 6 entrypoints"]
        gateway["gateway"]
        postsvc["postsvc"]
        authsvc["authsvc"]
        ac["analytics-consumer"]
        rw["reanalysis-worker"]
        nc["notification-consumer"]
    end
    subgraph internal["internal/"]
        direction TB
        abac_["abac — policy engine, JWT, demo users"]
        adapt_["adapt — Thrift ⇄ domain model + error mapping"]
        api_["api — REST handlers + router"]
        bootstrap_["bootstrap — storage/migration wiring"]
        cache_["cache — Redis + in-memory"]
        cli_["cli — Cobra CLI/REPL + serve.go"]
        export_["export — JSON/CSV writers"]
        handlers_["handlers — web UI + /web/* JSON API"]
        messaging_["messaging — kafka/, rabbitmq/, rocketmq/"]
        metrics_["metrics — Prometheus collectors"]
        middleware_["middleware — ABAC, rate limit, compression, ..."]
        ml_["ml/triton — KServe v2 HTTP client"]
        objectstore_["objectstore — MinIO wrapper"]
        rpcclient_["rpcclient — Kitex client ⇄ plain Go interfaces"]
        service_["service — PostService business logic"]
        storage_["storage — Postgres + file backends"]
    end
    subgraph idl["idl/"]
        thrift_["thrift/ — PostService, AuthService"]
        proto_["proto/ — Kafka/RocketMQ payloads"]
    end
    subgraph deploy["deployments/"]
        k8s_["k8s/ — Kustomize + Rollouts + Istio"]
        tf_["terraform/ — AWS · Azure · OCI"]
        infra_["nginx/, grafana/, triton/, rocketmq/"]
    end

    cmd --> internal
    internal --> idl
    cmd -.-> deploy
```

```
kitex_gen/                generated Kitex/Thrift code (committed)
assets/                   dashboard.html, home.html static assets, UI screenshots
postman/                  Postman collection + environment
api-docs.yaml             OpenAPI 3.0.3 spec
docker-compose.yml        full local stack (17 containers)
Dockerfile                single parameterized multi-stage build (ARG SERVICE=...)
.dockerignore             keeps build contexts to "what a Go build needs" (no .git, no Terraform state)
index.html, sitemap.xml   static marketing/landing page for the project (not served by the app itself)
Makefile
```

## Configuration reference

Everything is environment-variable driven — the full, authoritative list lives in `.env.example`. Every optional integration (Redis, each message broker, MinIO, Triton) degrades gracefully when disabled or unreachable: the service logs a warning and the dependent feature returns a clear error/`503` instead of failing startup.

<details>
<summary><strong>Server &amp; environment</strong></summary>

| Variable | Default | Purpose |
|---|---|---|
| `PORT` / `HOST` | `8080` / `0.0.0.0` | Gateway listen address |
| `ENVIRONMENT` | `development` | `development`, `staging`, or `production` |
| `READ_TIMEOUT` / `WRITE_TIMEOUT` / `IDLE_TIMEOUT` | `15s` / `15s` / `60s` | HTTP server timeouts |
| `SHUTDOWN_TIMEOUT` | `30s` | Grace period for in-flight requests during shutdown |

</details>

<details>
<summary><strong>Database</strong></summary>

| Variable | Default | Purpose |
|---|---|---|
| `DB_TYPE` | `file` | `file` (flat JSON, zero setup) or `postgres` |
| `DB_FILE_PATH` | `posts.json` | Used when `DB_TYPE=file` |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | `localhost` / `5432` / `postgres` / `postgres` / `postanalyzer` | Used when `DB_TYPE=postgres` — point this at **any** local Postgres |
| `DB_SSL_MODE` | `disable` | Postgres SSL mode |
| `DB_MAX_CONNS` / `DB_MIN_CONNS` | `25` / `5` | Connection pool bounds |

</details>

<details>
<summary><strong>Security &amp; auth</strong></summary>

| Variable | Default | Purpose |
|---|---|---|
| `JWT_SECRET` | `dev-only-change-me-in-production` | **Set a real value in production.** |
| `JWT_TTL` | `24h` | Token lifetime |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | `admin` / `admin123` | Seeded admin account (editor/viewer demo accounts always seeded too) |
| `RATE_LIMIT_REQUESTS` / `RATE_LIMIT_WINDOW` | `100` / `1m` | Per-client rate limit |
| `MAX_BODY_SIZE` | `1048576` (1MB) | Request body size cap |
| `ALLOWED_ORIGINS` | `*` | CORS — comma-separated or `*` |
| `TRUSTED_PROXIES` | _(empty)_ | Comma-separated IPs trusted to set `X-Forwarded-For` |

</details>

<details>
<summary><strong>Logging</strong></summary>

| Variable | Default | Purpose |
|---|---|---|
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | `json` or `text` |
| `LOG_OUTPUT` | `stdout` | `stdout` or a file path |
| `LOG_TIME_FORMAT` | `2006-01-02T15:04:05Z07:00` | Go time-format layout |

</details>

<details>
<summary><strong>RPC &amp; external HTTP</strong></summary>

| Variable | Default | Purpose |
|---|---|---|
| `POSTSVC_ADDR` / `AUTHSVC_ADDR` | `127.0.0.1:9001` / `127.0.0.1:9002` | Kitex RPC targets |
| `RPC_MUX` | `true` | Kitex connection multiplexing (one TCP connection, many concurrent calls) |
| `JSONPLACEHOLDER_URL` | `https://jsonplaceholder.typicode.com/posts` | Source for "Fetch Sample Posts" |
| `HTTP_TIMEOUT` | `30s` | Timeout for that outbound call |

</details>

<details>
<summary><strong>Redis, Kafka, RabbitMQ, RocketMQ, MinIO, Triton (all optional)</strong></summary>

| Variable | Default | Purpose |
|---|---|---|
| `REDIS_ENABLED` / `REDIS_ADDR` / `REDIS_PASSWORD` / `REDIS_DB` | `true` / `127.0.0.1:6379` / _(empty)_ / `0` | Cache — falls back to in-memory automatically if unreachable |
| `KAFKA_ENABLED` / `KAFKA_BROKERS` | `false` / `127.0.0.1:9092` | Event log producer |
| `RABBITMQ_ENABLED` / `RABBITMQ_URL` | `false` / `amqp://guest:guest@127.0.0.1:5672/` | Reanalysis work queue |
| `ROCKETMQ_ENABLED` / `ROCKETMQ_NAMESRV_ADDRS` | `false` / `127.0.0.1:9876` | Delayed notification delivery |
| `MINIO_ENABLED` / `MINIO_ENDPOINT` / `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY` / `MINIO_USE_SSL` | `false` / `127.0.0.1:9000` / `minioadmin` / `minioadmin` / `false` | Export persistence |
| `TRITON_ENABLED` / `TRITON_URL` | `false` / `http://127.0.0.1:8000` | ML sentiment classification |

</details>

## Testing

```bash
make test               # unit tests (race detector + coverage), no external dependencies
make test-integration   # storage integration tests against a real local Postgres
make postman-run        # Postman/newman collection against a running stack (BASE_URL=http://localhost by default)
make lint                # golangci-lint (0 issues)
make check                # vet + lint + test + gosec — full pre-commit gate
```

Representative per-package unit-test coverage (`go test ./... -cover`), all with `-race` clean and `golangci-lint` at 0 issues:

| Package | Coverage | Notes |
|---|---|---|
| `internal/errors`, `internal/logger` | 100% | Pure logic, fully deterministic |
| `internal/adapt` | ~87% | Includes the typed-error ⇄ `BaseResp` round trip |
| `internal/handlers` | ~84% | Web UI handlers + the `/web/*` JSON API |
| `internal/ml/triton` | ~89% | Against a fake KServe v2 HTTP server |
| `internal/api` | ~76% | REST handlers, via fake RPC clients |
| `internal/middleware` | ~78% | Includes gzip round-trip correctness (compress → decompress → byte-identical) |
| `internal/rpcclient` | Partial | Pure-function error-mapping logic (`rpcErr`) is unit-tested; the Kitex-dialing code itself is exercised via the live docker-compose stack, not mocked |

Integration- and infrastructure-dependent code (`internal/messaging/*`, `internal/objectstore`, the RPC dial paths, `cmd/*` wiring) is deliberately verified against the **real** local stack (`docker compose up`) rather than mocked — see [Testing](#testing) commands above and the CI workflow for how that's automated.

## Deployment

- **Docker Compose** — `make docker-up` / `docker compose up -d`, the full local stack.
- **Kubernetes** — `deployments/k8s/base` (Kustomize) with `overlays/{dev,staging,prod,local-kind}`. `make k8s-apply-dev` etc. Includes an Ingress, a `prod` overlay with PodDisruptionBudget + HorizontalPodAutoscaler, Argo Rollouts manifests for canary/blue-green (`deployments/k8s/rollout/`), and Istio mesh manifests (`deployments/k8s/mesh/`).
- **Local kind cluster** — `make kind-create && make kind-load && make k8s-apply-local-kind`.
- **Terraform (AWS / Azure / OCI)** — `deployments/terraform/environments/{aws,azure,oci}`, each provisioning the managed equivalent of the local stack (managed Kubernetes, managed Postgres, managed Redis, container registry, optional CDN). `make tf-validate` validates all three against their real provider schemas. Not applied by default — no cloud credentials are used by this repo; see each environment's `*.tfvars.example`.

See [ARCHITECTURE.md](./ARCHITECTURE.md#deployment-topology) for how the same six service images flow into all three targets, and how the Kubernetes/Terraform artifacts were actually validated (a real local `kind` cluster, `terraform validate` against live provider schemas).

## API documentation

- **OpenAPI 3.0.3**: `api-docs.yaml` (validated with `openapi-spec-validator`) — covers `/health`, `/readiness`, `/metrics`, `/api/v1/auth/login`, `/api/v1/posts` (list/create), `/api/v1/posts/{id}` (get/update/delete), `/api/v1/posts/bulk`, `/api/v1/posts/export`, `/api/v1/posts/analytics`, `/api/v1/posts/reanalyze`, `/api/v1/exports`, `/api/v1/exports/{key}`, `/api/v1/ml/sentiment`, `/api/v1/admin/status`.
- **Postman collection**: `postman/Post-Analyzer.postman_collection.json` + `postman/Post-Analyzer.postman_environment.json` — 27 requests covering CRUD, bulk ops, export, analytics, ABAC role checks (viewer/editor/admin), ML sentiment, and object-storage export listing/download. Runs clean via `make postman-run`.

## Makefile reference

Run `make help` for the live list. Grouped summary:

| Group | Targets |
|---|---|
| Build | `build`, `build-all`, `run`, `repl`, `dev` |
| Codegen | `thrift`, `proto`, `generate` |
| Test &amp; quality | `test`, `test-integration`, `test-coverage`, `bench`, `lint`, `vet`, `format`, `security`, `check` |
| API validation | `postman-run`, `openapi-validate` |
| Docker | `docker-build`, `docker-up`, `docker-down`, `docker-logs`, `docker-restart`, `docker-ps`, `docker-scale-gateway` |
| Kubernetes | `k8s-apply-dev`, `k8s-apply-staging`, `k8s-apply-prod`, `k8s-apply-local-kind`, `k8s-delete`, `k8s-status`, `k8s-rollout-status`, `k8s-rollback` |
| kind | `kind-create`, `kind-load`, `kind-delete` |
| Terraform | `tf-init`, `tf-plan`, `tf-validate` |
| Misc | `migrate`, `db-shell`, `clean`, `version`, `install`, `install-tools`, `init` |

## License

Distributed under the MIT License. See `LICENSE` for details.

---

Created by [Son Nguyen](https://github.com/hoangsonww).
