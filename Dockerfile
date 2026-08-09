# Shared multi-service build. SERVICE selects which cmd/<service> binary
# to build; every service in this repo (gateway, postsvc, authsvc,
# analytics-consumer, reanalysis-worker, notification-consumer) uses this
# same Dockerfile via docker-compose's `build.args`, so there's one build
# definition to maintain instead of six near-identical ones.
ARG SERVICE=gateway

# Build stage
FROM golang:1.25-alpine AS builder
ARG SERVICE
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags="-w -s" -o /build/service ./cmd/${SERVICE}

# Final stage
FROM alpine:latest
ARG SERVICE

RUN apk --no-cache add ca-certificates tzdata curl && \
    addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder /build/service ./service

# Only the gateway serves the web UI/static assets; harmless (small) to
# include for every service image, keeps this Dockerfile single-source.
COPY home.html ./
COPY assets ./assets

RUN mkdir -p /app/data && chown -R appuser:appuser /app

USER appuser

# Gateway HTTP; postsvc/authsvc RPC; consumers' /metrics. Only the
# relevant port is actually used per service — EXPOSE is documentation,
# not enforcement.
EXPOSE 8080 9001 9002 9101 9102 9103

CMD ["./service"]
