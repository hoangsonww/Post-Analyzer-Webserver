# Contributing to Post Analyzer Webserver

Thanks for taking the time to contribute. Please read this guide before opening a PR or issue.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Branching and Commits](#branching-and-commits)
- [Pull Requests](#pull-requests)
- [Testing](#testing)
- [Reporting Bugs](#reporting-bugs)
- [Requesting Features](#requesting-features)

---

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to uphold it.

---

## Getting Started

### Prerequisites

- Go 1.25+
- Docker + Docker Compose
- `make`

### Setup

**Option A — Docker Compose (recommended, brings up the full stack):**

```bash
git clone https://github.com/hoangsonww/Post-Analyzer-Webserver.git
cd Post-Analyzer-Webserver
make docker-up   # or: docker compose up -d
```

**Option B — Running a single service natively (needs Postgres/Redis/etc. reachable separately):**

```bash
go run ./cmd/gateway
```

The gateway is now at `http://localhost:8080` (via nginx on `:80` in Compose).

---

## Development Workflow

The repo is a **Go microservices monorepo** with one binary per service under `cmd/`:

| Service | Path | Description |
|---|---|---|
| `gateway` | `cmd/gateway` | HTTP/RPC entrypoint, web UI, CLI/REPL |
| `postsvc` | `cmd/postsvc` | Post CRUD + analytics, over Kitex RPC |
| `authsvc` | `cmd/authsvc` | Auth/JWT/ABAC, over Kitex RPC |
| `analytics-consumer` | `cmd/analytics-consumer` | Kafka consumer for analytics fan-out |
| `reanalysis-worker` | `cmd/reanalysis-worker` | RabbitMQ/RocketMQ consumer for scheduled rechecks |
| `notification-consumer` | `cmd/notification-consumer` | Notification fan-out consumer |

Shared logic lives under `internal/` (handlers, service layer, storage, messaging, RPC clients, ML/Triton integration, middleware, CLI). See [ARCHITECTURE.md](../ARCHITECTURE.md) for diagrams of how these fit together.

**Adding a new HTTP endpoint:** add a handler in `internal/api`, wire it in `cmd/gateway`, and add a service method in `internal/service` if it touches business logic.

**Adding a new CLI/REPL command:** add a Cobra subcommand under `internal/cli`, register it in both `internal/cli/root.go` and `internal/cli/repl.go`'s command tree builder.

---

## Branching and Commits

- Branch off `master`. Use a short, descriptive branch name:
  - `feat/reanalysis-backoff`
  - `fix/sentiment-empty-body`
  - `docs/architecture-diagram`
  - `chore/upgrade-kitex`

- Commit messages should be concise and use the imperative mood:
  - `add retry backoff to reanalysis worker`
  - `fix sentiment endpoint crashing on empty body`
  - `update architecture diagram for messaging fan-out`

- Do not commit directly to `master`.

---

## Pull Requests

- Fill out the PR template completely.
- Keep PRs focused — one logical change per PR.
- All PRs require passing CI (lint, tests, build, Docker image build, security scan).
- Add screenshots for any web UI or CLI/REPL output changes.
- Request review from a maintainer when ready.

**Before submitting:**

```bash
golangci-lint run ./...
go build ./...
go test -race ./...
```

---

## Testing

```bash
go test -v -race ./...                                # unit tests
go test -tags=integration -v ./internal/storage/...    # integration tests (needs a real Postgres — see docker-compose.yml)
```

**Rules:**
- Write tests for every function added or modified.
- Integration- and infrastructure-dependent code (messaging, object storage, RPC dial paths) is verified against the **real** local stack (`docker compose up`), not mocked — see [README.md](../README.md#testing).
- All tests must pass before a PR can be merged.

---

## Reporting Bugs

Use the [Bug Report](.github/ISSUE_TEMPLATE/bug_report.yml) issue template. Include:

- Steps to reproduce
- Expected vs. actual behavior
- Which service is affected (gateway, postsvc, authsvc, a consumer/worker, CLI/REPL, web UI)
- Relevant logs or screenshots

---

## Requesting Features

Use the [Feature Request](.github/ISSUE_TEMPLATE/feature_request.yml) issue template. Explain the problem you're solving, not just the solution you want.
