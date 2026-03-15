# System Specification: Core Banking Platform (Event-Driven Microservices)

## 1. Project Overview
The `core-banking-platform` is a highly scalable, distributed enterprise system representing the evolution of a monolithic banking core. It implements a Microservices Architecture driven by Apache Kafka, adhering strictly to the Database-per-Service pattern. The system demonstrates mastery of distributed data consistency, asynchronous event choreography, and robust financial precision without relying on managed Cloud Service Providers.

## 2. Technology Stack

| Component | Technology | Purpose |
| :--- | :--- | :--- |
| **Language** | Go 1.23+ | Core development utilizing goroutines, interfaces, and structured error handling. |
| **HTTP Router** | chi (net/http) | Lightweight, idiomatic HTTP router for all microservices; no framework lock-in. |
| **API Gateway** | Custom Go service (chi + net/http) | Single entry point, reverse proxy routing, and cross-cutting concerns (rate limiting). |
| **Databases** | PostgreSQL (Multiple) | Dedicated relational databases per microservice to ensure loose coupling. |
| **DB Driver** | pgx | High-performance, idiomatic Go PostgreSQL driver and connection pool. |
| **Migrations** | golang-migrate | Database schema versioning executed per microservice independently. |
| **Caching/KV** | Redis (go-redis) | Distributed caching, idempotency keys, and distributed locking. |
| **Event Streaming**| Apache Kafka (franz-go) | Backbone for asynchronous inter-service communication and event choreography. |
| **Testing** | Go testing, testify, Testcontainers-go | Unit testing and ephemeral container-based integration testing per service. |
| **Observability**| Prometheus (prometheus/client_golang), Grafana | Distributed metrics collection and custom dashboards. |
| **Tracing** | OpenTelemetry Go SDK / Zipkin exporter | Distributed request tracing across multiple microservices (W3C Trace Context). |
| **Documentation**| OpenAPI (Swagger via swaggo/swag) | Distributed API documentation aggregated at the Gateway level. |
| **Deployment** | Docker & Docker Compose | Containerization and complex local orchestration of the entire ecosystem. |

## 3. Core Architectural Patterns

* **Hexagonal Architecture (Ports and Adapters):** Applied strictly within *each* microservice to keep business logic isolated from infrastructure (Kafka, Postgres, HTTP). Ports are expressed as Go interfaces; adapters are concrete struct implementations.
* **Database-per-Service:** Each microservice owns its data. Direct database access from outside the owning service is strictly prohibited.
* **Event-Driven Architecture (EDA):** State changes are broadcasted as domain events via Kafka, allowing decoupled services to react.
* **Choreography-based Saga Pattern:** Distributed transactions (e.g., money transfers requiring updates in multiple accounts) are handled via asynchronous event reactions rather than a central orchestrator.
* **Transactional Outbox Pattern:** Guarantees reliable event publishing. Domain events are saved in the same local Postgres database transaction as the business entity, then relayed to Kafka to prevent dual-write inconsistencies.

## 4. Microservices Topology

The platform is divided into distinct, independently deployable bounded contexts:

1. **`api-gateway`:**
   * Routes incoming HTTP traffic to the appropriate backend services using a reverse proxy (`httputil.ReverseProxy`).
   * Handles global rate limiting using Redis.
2. **`account-service`:**
   * **Domain:** Account lifecycle, balances, and customer ledger.
   * **Storage:** Dedicated `account_db` (Postgres).
   * **Responsibilities:** Creates accounts, reserves funds, commits or rolls back balance changes based on Kafka events.
3. **`transfer-service`:**
   * **Domain:** Transaction intent, validation, and distributed orchestration.
   * **Storage:** Dedicated `transfer_db` (Postgres) + Redis (Idempotency).
   * **Responsibilities:** Receives transfer requests, validates idempotency, saves the intent, and publishes `TransferRequestedEvent` to Kafka to initiate the Saga.
4. **`notification-service`:**
   * **Domain:** Customer communication.
   * **Storage:** Dedicated `notification_db` (Postgres) for audit logs.
   * **Responsibilities:** Listens to Kafka for successful/failed transfers and simulates sending emails/SMS.

## 5. Domain Rules & Financial Precision

* **Strict Financial Types:** All monetary values use `github.com/shopspring/decimal` (`decimal.Decimal`) to guarantee arbitrary-precision arithmetic without floating-point errors.
* **Banker's Rounding:** Computations strictly apply `decimal.DivisionPrecision` with `HALF_EVEN` rounding mode via `shopspring/decimal` configuration.
* **Idempotency:** The `transfer-service` mandates an `Idempotency-Key` HTTP header. Redis stores the key and the response payload for 24 hours to safely replay identical requests without side effects.
* **Optimistic Concurrency:** Database rows include a `version` integer column. Updates use `WHERE id = $1 AND version = $2` and check `rows affected == 1` to prevent lost updates during concurrent high-frequency transfers.

## 6. Distributed Observability

In a microservices ecosystem, finding the root cause of an error is complex.
* **Correlation IDs:** Every incoming request at the `api-gateway` gets a unique `traceId` injected via chi middleware into the request context.
* **Log Aggregation & Tracing:** The `traceId` propagates through HTTP headers and Kafka record headers. Zipkin visualizes the entire journey of a transfer request across the Gateway, Transfer Service, Kafka, Account Service, and Notification Service.

## 7. Ecosystem Run Instructions
The entire distributed platform, including all infrastructure and microservices, is orchestrated locally:
`docker compose up -d --build`
This provisions: Zookeeper, Kafka, 3x PostgreSQL instances, Redis, Zipkin, Prometheus, Grafana, API Gateway, and the 3 Core Microservices.