# Go Feature-Based Modular Clean Architecture Boilerplate

A standard, production-ready Go microservice boilerplate implementing **Feature-Based Modular Clean Architecture** with Hexagonal Architecture (Ports & Adapters) principles and a Lightweight Domain-Driven Design (DDD) approach.

This boilerplate enforces strict separation of business logic from infrastructure, ensuring the application remains modular, highly testable, and independent of external technologies and frameworks.

---

## 📋 Table of Contents

- [Key Features](#-key-features)
- [Architecture & Design Principles](#-architecture--design-principles)
- [Project Structure](#-project-structure)
- [Prerequisites & Tooling](#-prerequisites--tooling)
- [Dependencies (Libraries & Frameworks)](#-dependencies-libraries--frameworks)
- [Quick Start](#-quick-start)
- [Environment Configuration (.env)](#-environment-configuration-env)
- [Makefile Commands](#-makefile-commands)
- [API Specifications & Endpoints](#-api-specifications--endpoints)
- [Code Generation](#-code-generation)
- [Testing & Code Quality](#-testing--code-quality)
- [Additional Documentation](#-additional-documentation)

---

## ✨ Key Features

- **Feature-First Architecture**: Business logic is organized around domain features (e.g., `internal/auth`, `internal/user`, `internal/health`) rather than monolithic global layers.
- **Dual Protocol Support**:
  - **gRPC Server** running on port `:50051`.
  - **HTTP REST Gateway** (powered by `grpc-gateway`) running on port `:8080`.
- **Dependency Injection (Google Wire)**: Compile-time, type-safe dependency injection without runtime reflection.
- **Swagger / OpenAPI 2.0**: Native Swagger UI embedded and served directly at `/swagger/`.
- **Authentication & Security**:
  - Complete JWT Access Token & Refresh Token lifecycle management.
  - Secure one-way password hashing using `bcrypt`.
- **Database & Caching**:
  - **PostgreSQL**: Primary relational database via GORM with connection pooling and automated schema migrations (`golang-migrate`).
  - **Modular Extensions (Redis & MongoDB)**: Pre-configured driver clients for Redis and MongoDB in `infrastructure/cache/redis` and `infrastructure/database/mongo`, ready to be wired into `bootstrap/wire.go` when features require caching or document storage.
- **Observability & Resilience**:
  - Structured JSON Logging (**Uber Zap**) correlated with OpenTelemetry trace & span IDs.
  - Prometheus metrics exposed via `/metrics`.
  - Refresh token hashing (SHA-256) & Token Reuse Compromise Detection (RFC 6819).
  - CORS middleware enabled on the HTTP gateway.
  - Graceful shutdown for both the HTTP gateway and gRPC server.
- **Container Ready**: Includes multi-stage build Dockerfile and `docker-compose.yml` for local development.

---

## 🏛 Architecture & Design Principles

This project combines Clean Architecture and Hexagonal Architecture:

```text
               HTTP (grpc-gateway) / gRPC
                        │
                        ▼
                     Handler
                        │
                        ▼
                     Usecase
                        │
                        ▼
             Repository / Gateway Interface
                │                  │
                ▼                  ▼
            Database         External Service
```

1. **Separation of Concerns**: Core business logic does not depend on HTTP frameworks, GORM, Redis, or gRPC transport. All interactions are decoupled through interfaces.
2. **Feature-First Organization**: Each feature directory under `internal/` packages its own `dto`, `entity`, `errors`, `handler`, `repository`, `usecase`, and `validator`.

---

## 📂 Project Structure

```text
.
├── api/
│   ├── external/          # External third-party proto contracts (client-only)
│   └── proto/             # Inbound Service Proto definitions (.proto) per feature
├── bootstrap/             # Dependency Injection (Wire) & Server Bootstrapping
│   ├── app.go             # Lifecycle & Server Runner (gRPC + HTTP)
│   ├── gateway.go         # HTTP REST Gateway setup & Swagger UI
│   ├── grpc.go            # gRPC Server setup & Interceptors
│   ├── logger.go          # Logger DI Provider Adapter
│   ├── wire.go            # Wire Injector Declaration
│   └── wire_gen.go        # Generated Wire Code
├── cmd/
│   └── api/
│       └── main.go        # Application main entrypoint
├── deployments/           # Dockerfile & Docker Compose configuration
│   ├── Dockerfile
│   └── docker-compose.yml
├── docs/                  # Detailed architectural documentation (architecture.md)
├── gen/                   # Code auto-generated from Proto & OpenAPI
│   ├── externalpb/        # Generated External Client stubs (No Gateway, No Swagger)
│   ├── openapi/           # Embedded OpenAPI JSON Spec (Inbound APIs only)
│   └── pb/                # Generated Inbound Go PB code & gRPC Gateway
├── infrastructure/        # External drivers & framework implementations
│   ├── cache/             # Redis Client
│   ├── config/            # Viper Config Loader
│   ├── database/          # GORM PostgreSQL & MongoDB connections
│   ├── external/          # Outbound External Microservice Adapters (gRPC / HTTP clients)
│   ├── middleware/        # gRPC Interceptors (Logging, Recovery, Auth)
│   ├── swagger/           # Embedded Swagger UI handler
│   └── telemetry/         # OpenTelemetry Tracer & Meter Providers
├── internal/              # Business Logic (Feature-First modular domains)
│   ├── auth/              # Authentication Feature (Login, Refresh, Logout)
│   ├── health/            # Health Check Probes (Liveness & Readiness)
│   └── user/              # User Management Feature (CRUD)
├── migrations/            # SQL database migration scripts
├── pkg/                   # Generic helpers & utilities (logger, crypto, pagination, validator, etc.)
└── scripts/               # Automation scripts (protogen.sh / protogen.bat)
```

---

## 🛠 Prerequisites & Tooling

Before running the application, make sure your system has the following software installed:

### 1. Core System Requirements
- **Go**: `v1.25` or later
- **Docker** & **Docker Compose**
- **GNU Make** (utility to run `Makefile` targets)
- **protoc** (Protocol Buffers Compiler v3+) — *optional, only required if modifying `.proto` files*

### 2. CLI Development Tools
To run all automation targets (`make migrate-*`, `make wire`, `make lint`, `make mocks`), install the following CLI tools:

```bash
# 1. Database Migration Tool (golang-migrate with postgres driver)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 2. Dependency Injection Generator (Google Wire)
go install github.com/google/wire/cmd/wire@latest

# 3. Standard Go Linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 4. Mock Generator for Unit Tests
go install go.uber.org/mock/mockgen@latest

# 5. Protoc Plugins (Only required when updating .proto schemas)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

---

## 📦 Dependencies (Libraries & Frameworks)

This boilerplate utilizes battle-tested third-party dependencies chosen for performance and reliability in production:

| Category | Package / Library | Version | Role & Usage |
| :--- | :--- | :--- | :--- |
| **Transport & API** | `google.golang.org/grpc` | `v1.83.0` | High-performance RPC server & client framework |
| | `github.com/grpc-ecosystem/grpc-gateway/v2` | `v2.29.0` | Reverse-proxy mapping gRPC services to HTTP RESTful JSON APIs |
| | `google.golang.org/protobuf` | `v1.36.11` | Protocol Buffers runtime encoder/decoder |
| **Dependency Injection** | `github.com/google/wire` | `v0.7.0` | Compile-time dependency injection without runtime reflection |
| **Database & Persistence** | `gorm.io/gorm` | `v1.31.2` | Developer-friendly ORM for database mapping |
| | `gorm.io/driver/postgres` | `v1.6.0` | PostgreSQL driver utilizing `pgx/v5` connection pooling |
| | `go.mongodb.org/mongo-driver` | `v1.17.9` | Official MongoDB driver for NoSQL document storage |
| **Caching** | `github.com/redis/go-redis/v9` | `v9.21.0` | High-performance Redis client for caching and rate-limiting |
| **Configuration** | `github.com/spf13/viper` | `v1.21.0` | Environment variables and `.env` configuration loader |
| **Observability** | `go.uber.org/zap` | `v1.28.0` | High-throughput structured logger with minimal memory allocations |
| | `go.opentelemetry.io/otel` | `v1.45.0` | Distributed telemetry standard (Distributed Tracing via OTLP) |
| | `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` | `v0.69.0` | Automated tracing interceptor for gRPC server |
| | `github.com/prometheus/client_golang` | `v1.24.1` | Prometheus metrics client exposed via `/metrics` |
| **Security & Auth** | `github.com/golang-jwt/jwt/v5` | `v5.3.1` | JWT Access & Refresh Token signing and verification (RFC 7519) |
| | `golang.org/x/crypto/bcrypt` | `v0.54.0` | One-way password hashing algorithm with adaptive work factor |
| **Data Validation** | `github.com/go-playground/validator/v10` | `v10.30.3` | Struct tag-based request payload validation |
| **Testing** | `github.com/stretchr/testify` | `v1.11.1` | Assertion toolkit and mocking suite for unit tests |

---

## 🚀 Quick Start

### 1. Clone & Setup Environment

Copy the example environment configuration file to `.env`:

```bash
cp .env.example .env
```

### 2. Start Local Infrastructure (Database, Redis, Jaeger)

Use Docker Compose to launch PostgreSQL, MongoDB, Redis, OTel Collector, and Jaeger:

```bash
make docker-up
```

### 3. Run Database Migrations

Apply database table migrations to PostgreSQL:

```bash
make migrate-up
```

### 4. Run the Application

Start the service locally:

```bash
make run
```

The application will listen on the following ports:
- **gRPC Server**: `localhost:50051`
- **HTTP Gateway**: `http://localhost:8080`
- **Swagger UI**: `http://localhost:8080/swagger/`

---

## ⚙️ Environment Configuration (.env)

Core settings are loaded from environment variables or the `.env` file:

| Variable | Default | Description |
| --- | --- | --- |
| `APP_NAME` | `go-feature-based-boilerplate` | Application name |
| `APP_ENV` | `development` | Environment (`development`, `production`, `test`) |
| `SERVER_GRPC_PORT` | `50051` | Port for the gRPC Server |
| `SERVER_HTTP_PORT` | `8080` | Port for the HTTP REST Gateway |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `appdb` | PostgreSQL database name |
| `DB_USER` | `appuser` | PostgreSQL username |
| `DB_PASSWORD` | `apppassword` | PostgreSQL password |
| `REDIS_ADDR` | `localhost:6379` | Redis Server address |
| `MONGO_URI` | `mongodb://localhost:27017` | MongoDB connection URI |
| `JWT_SECRET_KEY` | `change-me-...` | Secret key for JWT token signing |
| `JWT_ACCESS_TOKEN_TTL`| `15m` | Access token lifespan |
| `JWT_REFRESH_TOKEN_TTL`| `720h` | Refresh token lifespan |
| `LOG_LEVEL` | `debug` | Log level (`debug`, `info`, `warn`, `error`) |
| `OTEL_ENABLED` | `false` | Enable OpenTelemetry trace exporting |

---

## 🛠 Makefile Commands

This repository includes a comprehensive set of automated developer commands:

### Build & Run
- `make run` — Run application directly (`go run ./cmd/api`)
- `make build` — Compile production binary to `bin/server`

### Code Generation & DI
- `make protogen` — Generate Go protocol buffer code and OpenAPI JSON from `.proto` files
- `make wire` — Generate `bootstrap/wire_gen.go` using Google Wire
- `make mocks` — Generate mocks for unit testing (`go generate ./internal/...`)

### Database & Migrations
- `make migrate-up` — Apply all pending SQL migrations
- `make migrate-down` — Roll back 1 migration step
- `make migrate-status` — Check current database migration version
- `make migration name=<desc>` — Create a new SQL migration file pair

### Testing & Quality
- `make test` — Run all unit tests (`./internal/...` `./pkg/...`)
- `make test-cover` — Run unit tests and generate HTML coverage report (`coverage.html`)
- `make lint` — Run `golangci-lint`
- `make fmt` — Format Go source files (`gofmt` & `goimports`)

### Docker Operations
- `make docker-up` — Start all infrastructure services via docker-compose
- `make docker-down` — Stop and tear down infrastructure containers
- `make docker-logs` — Stream container logs from docker-compose

---

## 📡 API Specifications & Endpoints

### 🟢 Probes & Metrics (Public)
| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/healthz` | Kubernetes Liveness Probe (De-facto Standard) |
| `GET` | `/health` | Alternative Liveness Probe |
| `GET` | `/readyz` | Kubernetes Readiness Probe (Returns 503 if downstream dependencies fail) |
| `GET` | `/metrics` | Prometheus Metrics Endpoint |

### 🔐 Authentication (`/api/v1/auth`)
| Method | Endpoint | Description | Auth |
| --- | --- | --- | --- |
| `POST` | `/api/v1/auth/login` | Authenticate user & issue JWT token pair | Public |
| `POST` | `/api/v1/auth/refresh` | Renew Access Token using Refresh Token | Public |
| `POST` | `/api/v1/auth/logout` | Revoke Refresh Token & Logout | Public |

### 👤 User Management (`/api/v1/users`)
| Method | Endpoint | Description | Auth |
| --- | --- | --- | --- |
| `POST` | `/api/v1/users` | Register a new user | Public |
| `GET` | `/api/v1/users/{id}` | Retrieve user profile by ID | Bearer JWT |
| `PATCH` | `/api/v1/users/{id}` | Update user profile data | Bearer JWT |
| `DELETE` | `/api/v1/users/{id}` | Soft/hard delete user | Bearer JWT |

### 📄 Documentation & UI
- **Swagger UI**: Access interactively in the browser at `http://localhost:8080/swagger/`

---

## 🔄 Code Generation

### 1. Protobuf & OpenAPI
When you update or add new `.proto` schemas in `api/proto/`:
```bash
make protogen
```
*The script `scripts/protogen.sh` (or `protogen.bat` on Windows) automatically generates the Go protocol buffer structs in `gen/pb/` and OpenAPI JSON specs in `gen/openapi/`.*

### 2. Dependency Injection (Wire)
When you add new providers in `bootstrap/wire.go` or constructors in `internal/`:
```bash
make wire
```

---

## 🧪 Testing & Code Quality

Run unit tests:
```bash
make test
```

Generate a test coverage report:
```bash
make test-cover
```

Run linter to enforce code standards:
```bash
make lint
```

---

## 📚 Additional Documentation

For an in-depth technical explanation of architectural decisions, Clean Architecture rules, handler registration order, and error handling patterns, refer to:
- [docs/architecture.md](docs/architecture.md) (English Documentation)
- [docs/architecture.id.md](docs/architecture.id.md) (Dokumentasi Bahasa Indonesia)
