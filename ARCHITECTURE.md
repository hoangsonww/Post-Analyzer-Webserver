# Architecture

This complements the [README](./README.md)'s ASCII overview with diagrams for the pieces that are easier to follow as a picture: the full component graph, the request lifecycle through ABAC and RPC, the post-creation event fan-out across three message brokers, and how the same six service images map onto Docker Compose vs. Kubernetes vs. the Terraform cloud modules.

## System components

```mermaid
flowchart TB
    Browser(["Browser<br/>(UI + Dashboard)"])

    subgraph Edge["Edge"]
        Nginx["nginx<br/>(resolver-based LB —<br/>scales with gateway)"]
    end

    subgraph GW["gateway (Hertz / Netpoll)"]
        MW["Middleware:<br/>RequestID → Logging → Recovery →<br/>SecurityHeaders → CORS → RateLimit →<br/>MaxBodySize → Timeout → Compression → Metrics"]
        ABAC["ABAC middleware<br/>(per-route, JWT + policy check)"]
        API["REST handlers<br/>(/api/v1/*)"]
    end

    subgraph RPC["Kitex RPC (Thrift, TTHeader, mux)"]
        PostSvc["postsvc<br/>CRUD, analysis, cache invalidation"]
        AuthSvc["authsvc<br/>JWT issuance, ABAC policy decisions"]
    end

    subgraph Data["Storage"]
        PG[("Postgres<br/>(or file store)")]
        Redis[("Redis cache<br/>(memory fallback)")]
        MinIO[("MinIO<br/>(S3-compatible exports)")]
    end

    subgraph Messaging["Messaging — three brokers, three distinct jobs"]
        Kafka["Kafka<br/>post.events<br/>(durable event log)"]
        RabbitMQ["RabbitMQ<br/>reanalysis.jobs<br/>(work queue)"]
        RocketMQ["RocketMQ<br/>post-notifications<br/>(native delayed delivery)"]
    end

    subgraph Consumers["Consumers"]
        Analytics["analytics-consumer"]
        Reanalysis["reanalysis-worker"]
        Notification["notification-consumer"]
    end

    Triton["Nvidia Triton<br/>(ONNX sentiment model)"]
    Obs["Prometheus → Grafana<br/>(every service's /metrics)"]

    Browser --> Nginx --> MW --> ABAC --> API
    API -->|Kitex RPC| PostSvc
    API -->|Kitex RPC| AuthSvc
    PostSvc --> PG
    PostSvc --> Redis
    PostSvc --> MinIO
    PostSvc -->|publish| Kafka
    PostSvc -->|publish, delayed| RocketMQ
    PostSvc -->|classify on create| Triton
    API -->|enqueue| RabbitMQ

    Kafka --> Analytics
    RabbitMQ --> Reanalysis
    RocketMQ --> Notification
    Reanalysis -->|Kitex RPC| PostSvc

    PostSvc -.-> Obs
    AuthSvc -.-> Obs
    GW -.-> Obs
    Analytics -.-> Obs
    Reanalysis -.-> Obs
    Notification -.-> Obs
```

## Request lifecycle: an authenticated write

`PUT /api/v1/posts/{id}` end to end — every mutating request goes through the same ABAC check; the diagram below is representative of any protected route, not special-cased for this one.

```mermaid
sequenceDiagram
    participant C as Client
    participant N as nginx
    participant MW as Gateway middleware chain
    participant ABAC as ABAC middleware
    participant Auth as authsvc (RPC)
    participant API as REST handler
    participant Post as postsvc (RPC)
    participant Cache as Redis
    participant DB as Postgres

    C->>N: PUT /api/v1/posts/42<br/>Authorization: Bearer <jwt>
    N->>MW: proxy (X-Forwarded-*)
    MW->>MW: RequestID, Logging, Recovery,<br/>RateLimit, MaxBodySize, Timeout
    MW->>ABAC: request + JWT
    ABAC->>Auth: ValidateToken(jwt)
    Auth-->>ABAC: Subject{role, username}
    ABAC->>Auth: Authorize(Request{resource:"post", action:"write", subject})
    Auth-->>ABAC: Decision (policy eval, default-deny)
    alt denied
        ABAC-->>C: 401/403
    else allowed
        ABAC->>API: forward, Subject in context
        API->>Post: UpdatePost(id, fields) [Kitex/Thrift/TTHeader]
        Post->>DB: UPDATE posts SET ... WHERE id = ?
        Post->>Cache: invalidate posts:42, posts:all
        Post-->>API: updated Post
        API-->>C: 200 {"data": {...}, "meta": {...}}
    end
```

## Post creation: fan-out across three brokers

This is the one flow that touches every messaging integration at once, which is why it's worth its own diagram — `PublishPostEvent` and `PublishScheduledRecheck` are both fire-and-forget from the caller's perspective (errors are logged, never fail the CRUD operation that triggered them).

```mermaid
sequenceDiagram
    participant API as gateway
    participant Post as postsvc
    participant Triton as Triton (ML)
    participant DB as Postgres
    participant Kafka as Kafka (post.events)
    participant RMQ as RocketMQ (post-notifications)
    participant AC as analytics-consumer
    participant NC as notification-consumer

    API->>Post: CreatePost(title, body, userId) [RPC]
    Post->>DB: INSERT INTO posts ... RETURNING id
    Post-->>API: created Post
    API-->>API: 201 response returned to client immediately

    par Kafka event (durable, replayable)
        Post->>Triton: ClassifySentiment(title + body)
        Triton-->>Post: {label, probabilities}
        Post->>Kafka: publish PostEvent{type:"created", sentiment}
        Kafka->>AC: consume
        AC->>AC: increment post_events_consumed_total{event_type}
    and RocketMQ delayed notification (~10s)
        Post->>RMQ: publish ScheduledNotification (delay level: 10s)
        Note over RMQ: broker holds the message,<br/>delivers after the delay — no polling,<br/>no external scheduler needed
        RMQ->>NC: deliver after ~10s
        NC->>NC: log "notification delivered"
    end
```

RabbitMQ isn't in this diagram because it belongs to a different flow: `POST /api/v1/posts/reanalyze` enqueues a job onto `reanalysis.jobs`, and `reanalysis-worker` — a competing consumer, not a fan-out subscriber — dequeues and processes it by calling back into `postsvc` over RPC. Three brokers, three different delivery semantics, each matched to what it's actually for (see the README's [Messaging](./README.md#messaging-why-three-brokers) section).

## Deployment topology

The same six service images flow into three different deployment targets. Docker Compose is what's actually run and tested in this repo; Kubernetes and Terraform are structured and validated so a real deployment is a matter of applying them, not designing them from scratch later.

```mermaid
flowchart LR
    subgraph Images["Shared Dockerfile (ARG SERVICE=...)"]
        I1["gateway"]
        I2["postsvc"]
        I3["authsvc"]
        I4["analytics-consumer"]
        I5["reanalysis-worker"]
        I6["notification-consumer"]
    end

    subgraph Compose["docker-compose.yml — local, tested"]
        direction TB
        C1["16 containers:<br/>6 services + Postgres, Redis, Kafka,<br/>RabbitMQ, RocketMQ×2, MinIO, Triton,<br/>nginx, Prometheus, Grafana"]
    end

    subgraph K8s["deployments/k8s — Kustomize"]
        direction TB
        K1["base/ + overlays:<br/>dev · staging · prod · local-kind"]
        K2["rollout/ — Argo Rollouts<br/>(canary / blue-green)"]
        K3["mesh/ — Istio Gateway/VirtualService"]
        K1 --> K2
        K1 --> K3
    end

    subgraph TF["deployments/terraform — validated, not applied"]
        direction TB
        T1["AWS: EKS · RDS · ElastiCache · ECR ·<br/>CloudFront (optional)"]
        T2["Azure: AKS · Postgres Flexible ·<br/>Redis Cache · ACR · Front Door (optional)"]
        T3["OCI: OKE · PSQL · OCIR<br/>(no first-party CDN — documented gap)"]
    end

    Images --> Compose
    Images --> K8s
    K8s -.->|provisions the cluster K8s runs on| TF
```

Kubernetes and the Terraform modules were validated directly, not just written and assumed correct: the K8s overlays were applied to a real local `kind` cluster (full auth+CRUD through an installed `ingress-nginx`, a live `kubectl rollout` + `rollout undo`), and each Terraform module was checked with `terraform validate` against its real provider schema (`terraform providers schema -json`), which is what caught real mismatches like OCI's `oci_psql_db_system` needing `subnet_id`/`availability_domain` nested inside a `network_details {}` block rather than at the top level.
