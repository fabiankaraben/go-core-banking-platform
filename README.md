# Go Core Banking Platform

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/doc/go1.23)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Architecture](https://img.shields.io/badge/Architecture-Hexagonal-orange)](https://alistair.cockburn.us/hexagonal-architecture/)
[![Event Driven](https://img.shields.io/badge/Pattern-Event--Driven-blueviolet)](https://martinfowler.com/articles/201701-event-driven.html)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](docker-compose.yml)
[![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL-336791?logo=postgresql)](https://www.postgresql.org/)
[![Kafka](https://img.shields.io/badge/Broker-Apache%20Kafka-231F20?logo=apachekafka)](https://kafka.apache.org/)
[![Redis](https://img.shields.io/badge/Cache-Redis-DC382D?logo=redis)](https://redis.io/)
[![Prometheus](https://img.shields.io/badge/Metrics-Prometheus-E6522C?logo=prometheus)](https://prometheus.io/)
[![Grafana](https://img.shields.io/badge/Dashboards-Grafana-F46800?logo=grafana)](https://grafana.com/)
[![Zipkin](https://img.shields.io/badge/Tracing-Zipkin-FE7139)](https://zipkin.io/)

A production-grade, event-driven core banking platform built in Go, demonstrating real-world microservice patterns including **Hexagonal Architecture**, **Choreography-based Saga**, **Transactional Outbox**, **Optimistic Concurrency**, and **Distributed Observability**.

---

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Services](#services)
- [Technology Stack](#technology-stack)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [API Reference](#api-reference)
- [Running Tests](#running-tests)
- [Observability](#observability)
- [Design Decisions](#design-decisions)

---

## Overview

This platform implements a money transfer system composed of four loosely coupled microservices that communicate asynchronously via Apache Kafka. Each service owns its own PostgreSQL database (Database-per-Service pattern), and cross-service consistency is achieved through a **Choreography-based Saga** with a **Transactional Outbox** to guarantee at-least-once event delivery.

---

## Architecture

```
                        ┌─────────────────────────────────────────────┐
                        │               API Gateway :8080              │
                        │  Rate Limiting · Reverse Proxy · Correlation │
                        └────────────┬──────────────┬──────────────────┘
                                     │              │
                         ┌───────────▼──┐    ┌──────▼────────────┐
                         │   Account    │    │    Transfer       │
                         │  Service     │    │    Service        │
                         │   :8081      │    │     :8082         │
                         │  PostgreSQL  │    │  PostgreSQL+Redis │
                         └──────┬───────┘    └──────┬────────────┘
                                │                   │
                    ┌───────────▼───────────────────▼───────────┐
                    │              Apache Kafka                   │
                    │  transfers.requested / completed / failed   │
                    └───────────────────────┬───────────────────┘
                                            │
                              ┌─────────────▼──────────────┐
                              │   Notification Service       │
                              │         :8083                │
                              │      PostgreSQL              │
                              └─────────────────────────────┘
```

### Transfer Saga Flow

```
Client ──POST /api/v1/transfers──► API Gateway ──► Transfer Service
                                                        │
                                          (1) Persist Transfer (pending)
                                          (2) Write TransferRequested → Outbox
                                          (3) Outbox Relay publishes to Kafka
                                                        │
                                          ┌─────────────▼─────────────┐
                                          │      Account Service       │
                                          │  Debit source account      │
                                          │  Credit dest account       │
                                          │  Publish Completed/Failed  │
                                          └───────────────────────────┘
                                                        │
                                     ┌──────────────────┴──────────────────┐
                                     ▼                                       ▼
                             Transfer Service                    Notification Service
                           Update status → completed/failed     Log email + SMS notification
```

---

## Services

| Service | Port | Database | Responsibilities |
|---|---|---|---|
| **api-gateway** | 8080 | — | Reverse proxy, rate limiting (Redis), correlation IDs |
| **account-service** | 8081 | PostgreSQL (accounts_db) | Account CRUD, debit/credit with optimistic locking, saga participant |
| **transfer-service** | 8082 | PostgreSQL (transfers_db) + Redis | Transfer orchestration, idempotency (Redis), saga initiator |
| **notification-service** | 8083 | PostgreSQL (notifications_db) | Consumes completed/failed events and logs notifications |

---

## Technology Stack

| Layer | Technology |
|---|---|
| Language | Go 1.23+ |
| HTTP Router | [chi v5](https://github.com/go-chi/chi) |
| Database Driver | [pgx v5](https://github.com/jackc/pgx) |
| Migrations | [golang-migrate v4](https://github.com/golang-migrate/migrate) |
| Kafka Client | [franz-go](https://github.com/twmb/franz-go) |
| Redis Client | [go-redis v9](https://github.com/redis/go-redis) |
| Financial Precision | [shopspring/decimal](https://github.com/shopspring/decimal) |
| Logging | [zap](https://github.com/uber-go/zap) |
| Metrics | [Prometheus](https://prometheus.io/) + Grafana |
| Tracing | [OpenTelemetry](https://opentelemetry.io/) + Zipkin |
| Testing | [testify](https://github.com/stretchr/testify) |
| Infrastructure | Docker Compose |

---

## Project Structure

```
go-core-banking-platform/
├── docker-compose.yml          # Full infrastructure + services
├── prometheus.yml              # Prometheus scrape configuration
│
├── api-gateway/
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── config/config.go
│   │   └── middleware/middleware.go  # Rate limiting, correlation IDs
│   └── Dockerfile
│
├── account-service/
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── domain/             # Account aggregate, domain errors, events
│   │   ├── ports/              # Inbound & outbound interfaces
│   │   ├── app/                # Business logic (service.go + tests)
│   │   └── adapters/
│   │       ├── postgres/       # account_repo.go, outbox_repo.go
│   │       ├── kafka/          # producer, consumer, outbox_relay
│   │       └── http/           # handler.go
│   ├── migrations/             # SQL up/down migrations
│   └── Dockerfile
│
├── transfer-service/
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── domain/             # Transfer aggregate, events
│   │   ├── ports/              # Interfaces incl. IdempotencyStore
│   │   ├── app/                # Business logic + tests
│   │   └── adapters/
│   │       ├── postgres/       # transfer_repo.go, outbox_repo.go
│   │       ├── redis/          # idempotency.go
│   │       ├── kafka/          # producer, consumer, outbox_relay
│   │       └── http/           # handler.go
│   ├── migrations/
│   └── Dockerfile
│
├── notification-service/
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── domain/             # Notification entity, event types
│   │   ├── app/                # Service logic + tests
│   │   └── adapters/
│   │       ├── postgres/       # notification_repo.go
│   │       └── kafka/          # consumer.go
│   ├── migrations/
│   └── Dockerfile
│
└── system-specs/
    └── system-design.md
```

---

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) ≥ 24
- [Docker Compose](https://docs.docker.com/compose/) v2
- [Go](https://go.dev/) 1.23+ (for local development / tests)

### Start the Platform

```bash
# Clone the repository
git clone https://github.com/fabiankaraben/go-core-banking-platform.git
cd go-core-banking-platform

# Start all infrastructure and services
docker compose up --build
```

All services will be available after ~30 seconds (Kafka broker readiness is the typical bottleneck).

### Stop the Platform

```bash
docker compose down -v   # -v removes volumes (wipes data)
```

---

## API Reference

All requests go through the API Gateway at `http://localhost:8080/api/v1`.

### Accounts

#### Create Account

```http
POST /api/v1/accounts
Content-Type: application/json

{
  "customer_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "currency":    "USD",
  "balance":     "1000.00"
}
```

#### Get Account

```http
GET /api/v1/accounts/{accountID}
```

### Transfers

#### Initiate Transfer

```http
POST /api/v1/transfers
Content-Type: application/json
Idempotency-Key: <unique-client-generated-key>

{
  "source_account_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "dest_account_id":   "7cb9b63a-1234-4321-b3fc-abcdef123456",
  "amount":            "250.00",
  "currency":          "USD"
}
```

The `Idempotency-Key` header is **required** and must be unique per transfer request. Re-submitting the same key returns the original response without creating a duplicate transfer.

#### Get Transfer Status

```http
GET /api/v1/transfers/{transferID}
```

Transfer statuses: `pending` → `completed` | `failed`

---

## Running Tests

Each service has its own Go module. Unit tests use `testify` mocks and do not require any running infrastructure.

```bash
# account-service unit tests
cd account-service && go test ./internal/... -v -race

# transfer-service unit tests
cd transfer-service && go test ./internal/... -v -race

# notification-service unit tests
cd notification-service && go test ./internal/... -v -race

# Run all services' tests from repo root (requires Go 1.23+)
for svc in account-service transfer-service notification-service; do
  echo "=== Testing $svc ==="
  (cd $svc && go test ./internal/... -v -race -count=1)
done
```

### Test Coverage

```bash
cd account-service
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

---

## Observability

| Tool | URL | Credentials |
|---|---|---|
| Prometheus | http://localhost:9090 | — |
| Grafana | http://localhost:3000 | admin / admin |
| Zipkin | http://localhost:9411 | — |

### Metrics Endpoints

Each service exposes Prometheus metrics at `/metrics`:

- `http://localhost:8080/metrics` — API Gateway
- `http://localhost:8081/metrics` — Account Service
- `http://localhost:8082/metrics` — Transfer Service
- `http://localhost:8083/metrics` — Notification Service

### Health Checks

All services expose `GET /health` returning HTTP 200 when ready.

---

## Design Decisions

### Hexagonal Architecture (Ports & Adapters)

Each service separates business logic from infrastructure. Domain objects and `ports/` interfaces have zero external dependencies. Adapters (PostgreSQL, Kafka, Redis) implement those interfaces and are injected at startup.

### Transactional Outbox Pattern

Rather than publishing Kafka events directly during a database transaction (which risks split-brain on failure), each service writes events to an `outbox_events` table within the same transaction. A background `OutboxRelay` goroutine polls for unpublished events and publishes them to Kafka, then marks them published. This guarantees **at-least-once delivery**.

### Choreography-based Saga

No central orchestrator exists. The `transfer-service` publishes `TransferRequested`, the `account-service` reacts and publishes `TransferCompleted` or `TransferFailed`, and both the `transfer-service` and `notification-service` react to those outcome events. This achieves distributed consistency without a single point of failure.

### Optimistic Concurrency Control

Account and Transfer updates use a `version` column. Updates include `WHERE id = $1 AND version = $2` and bump the version. A zero-rows-affected result signals a concurrent modification, returning `ErrOptimisticLock` to the caller for retry.

### Idempotency

The `transfer-service` accepts an `Idempotency-Key` header. It checks Redis first (fast path) then the database. Duplicate keys return the original transfer response without side effects, making the API safe to retry.

### Financial Precision

All monetary values use `shopspring/decimal` — never `float64` — to avoid floating-point representation errors. Values are stored as `NUMERIC(20,8)` in PostgreSQL and serialized as strings in JSON.
