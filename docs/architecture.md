# Go Microservice Blueprint (Feature-Based Modular Clean Architecture)

> 🌐 **Language / Bahasa**: **English** | [Bahasa Indonesia](architecture.id.md)

Feature-Based Modular Clean Architecture combining Hexagonal Architecture principles (Ports & Adapters) and a Lightweight Domain-Driven Design (DDD) approach, isolating business logic from infrastructure to keep the application modular, testable, and technology-agnostic.

> **Objective**
>
> This blueprint serves as the engineering standard for all Go microservices built with:
>
> * Go
> * gRPC + grpc-gateway (REST via gateway)
> * Google Wire (Compile-Time Dependency Injection)
> * GORM
> * PostgreSQL
> * MongoDB
> * Docker
> * JWT
> * Unit Testing
> * Integration Testing
> * Observability (Full-Stack APM: Structured Logging, Metrics, Distributed Tracing)

---

# Design Principles

## 1. Separation of Responsibility

Each directory has a single, well-defined responsibility.

| Directory        | Responsibility                                 |
| ---------------- | ---------------------------------------------- |
| `cmd`            | Application Entry Point                        |
| `bootstrap`      | Dependency Injection & Application Composition |
| `internal`       | Business Logic (Domain Features)               |
| `infrastructure` | Framework & External Driver Implementation     |
| `pkg`            | Shared Library, Helper, Wrapper (Generic)      |
| `api`            | API Contract (Proto + OpenAPI)                 |
| `gen`            | Generated Code (Unmodified by hand)            |
| `configs`        | Configuration Files (YAML, `.env`)             |
| `migrations`     | Database Migrations                            |
| `tests`          | Integration & E2E Tests                        |

---

## 2. Clean Architecture

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
        Database          External Service
```

Business logic is completely decoupled from concrete transport and storage implementations:

* HTTP
* Database
* Redis
* Kafka
* gRPC
* External Services

The business core communicates exclusively through Interfaces (Ports).

---

## 3. Feature First

All business code is organized by business feature/domain:

```text
User
Auth
Order
Payment
Inventory
```

Organized by feature, not by global technical layers.

---

# Project Structure

```text
my-service/
│
├── cmd/
│   └── api/
│       └── main.go
│
├── bootstrap/
│   ├── app.go
│   ├── grpc.go
│   ├── gateway.go
│   ├── logger.go
│   ├── wire.go
│   └── wire_gen.go
│
├── api/
│   ├── proto/
│   │   ├── user/
│   │   │   └── v1/
│   │   │       └── user.proto
│   │   └── auth/
│   │       └── v1/
│   │           └── auth.proto
│   └── openapi/
│
├── gen/
│   ├── proto/
│   │   ├── user/
│   │   │   └── v1/
│   │   │       ├── user.pb.go
│   │   │       ├── user_grpc.pb.go
│   │   │       └── user.pb.gw.go
│   │   └── auth/
│   │       └── v1/
│   │           ├── auth.pb.go
│   │           ├── auth_grpc.pb.go
│   │           └── auth.pb.gw.go
│   └── openapi/
│
├── internal/
│   ├── auth/
│   │   ├── dto/
│   │   ├── entity/
│   │   ├── errors/
│   │   ├── handler/
│   │   ├── repository/
│   │   │   └── interface.go
│   │   ├── usecase/
│   │   ├── validator/
│   │   └── provider.go
│   │
│   ├── user/
│   │   ├── dto/
│   │   ├── entity/
│   │   ├── errors/
│   │   ├── handler/
│   │   ├── repository/
│   │   │   └── interface.go
│   │   ├── usecase/
│   │   ├── validator/
│   │   └── provider.go
│   │
│   ├── order/
│   ├── payment/
│   └── inventory/
│
├── pkg/
│   ├── pagination/
│   ├── errorutil/
│   ├── pointer/
│   ├── timeutil/
│   ├── stringutil/
│   ├── cryptoutil/
│   ├── jwtutil/
│   ├── contextutil/
│   ├── httpclient/
│   ├── idempotency/
│   ├── routine/
│   └── logger/
│
├── infrastructure/
│   ├── database/
│   │   ├── postgres/
│   │   └── mongo/
│   │
│   ├── cache/
│   │   └── redis/
│   │
│   ├── messaging/
│   │   ├── kafka/
│   │   └── rabbitmq/
│   │
│   ├── grpc/
│   │   └── client/
│   │
│   ├── http/
│   │   ├── payment/
│   │   ├── auth/
│   │   └── notification/
│   │
│   ├── storage/
│   │   └── s3/
│   │
│   ├── middleware/
│   ├── telemetry/
│   └── config/
│
├── configs/
│   ├── config.yaml
│   ├── config.dev.yaml
│   └── config.prod.yaml
│
├── migrations/
│
├── scripts/
│
├── deployments/
│   ├── Dockerfile
│   └── docker-compose.yml
│
├── tests/
│   ├── integration/
│   └── e2e/
│
├── Makefile
└── go.mod
```

---

# Dependency Rules

## cmd

Application Entry Point only.

May import:

* `bootstrap`

Must NOT import:

* `internal` directly
* `infrastructure` directly

---

## bootstrap

Composition Root.

Contains:

* Google Wire
* gRPC Server initialization
* grpc-gateway (REST) initialization
* Middleware Registration & Server Lifecycle Container

`bootstrap` is allowed to know about the entire project to assemble dependencies.

Because this is a microservice, each service has a focused feature scope. `bootstrap/grpc.go` and `bootstrap/gateway.go` will not grow uncontrollably. If a service begins to accumulate too many handlers, that is an architectural signal to split into separate microservices, rather than introducing routing layer complexity.

---

## internal

Contains all Business Logic.

Must NOT depend on:

* GORM
* Mongo Driver
* Redis Client
* Kafka
* HTTP Client
* gRPC Client
* Any third-party framework

The business layer only interacts with Domain Interfaces.

---

## infrastructure

Contains technical implementations and external drivers.

Examples:

* PostgreSQL
* MongoDB
* Redis
* Kafka
* S3
* SMTP
* Outbound HTTP/gRPC Clients

Must NOT contain Business Rules.

---

## pkg

Contains shared utilities, helpers, and generic wrappers used globally across all layers.

May be imported by:

* `internal`
* `infrastructure`
* `bootstrap`

Must NOT depend on:

* `internal` (business logic)
* `infrastructure` (framework implementations & configs)
* `bootstrap`

`pkg` must be **self-contained**, **generic**, and **stateless**. It must not hold dependencies on any layer above it.

---

## api

Contains API contract source definitions.

Examples:

* Protocol Buffers (.proto with grpc-gateway annotations)
* OpenAPI definitions (generated from proto)

Must NOT contain business logic.

---

## gen

Contains generated code only.

Must NEVER be edited manually.

Includes:

* `*.pb.go` (Protobuf messages)
* `*_grpc.pb.go` (gRPC server/client interfaces)
* `*.pb.gw.go` (grpc-gateway HTTP reverse proxy)
* Mock files (generated via mockgen/mockery)

---

## configs vs infrastructure/config

| Path                     | Content                                           |
| ------------------------ | ------------------------------------------------- |
| `configs/`               | Configuration files (YAML, `.env`, JSON)          |
| `infrastructure/config/` | Go code for loading, parsing, and validating configs |

Example `infrastructure/config/`:

```go
package config

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    Logger   LoggerConfig
}

func Load(path string) (*Config, error) {
    // viper or environment variable loading logic
}
```

---

# Feature Structure

Example User module:

```text
user/
├── dto/
│   ├── request.go
│   └── response.go
├── entity/
│   └── user.go
├── errors/
│   └── errors.go
├── handler/
│   └── handler.go
├── repository/
│   ├── interface.go
│   └── postgres/
│       └── repository.go
├── usecase/
│   ├── create.go
│   ├── create_test.go
│   └── find.go
├── validator/
│   └── validator.go
└── provider.go
```

## Feature Scaffolding Automation (`make new-feature`)

To accelerate development and guarantee architectural consistency across teams, the boilerplate provides an automated feature generator via Makefile:

```bash
make new-feature name=product
```

This command executes the shell script [scripts/new-feature.sh](file:///Users/a2375/Projects/go-feature-based-boilerplate/scripts/new-feature.sh) and automatically creates:
- Protobuf API contract file at `api/proto/product/product.proto`.
- Complete directory skeleton under `internal/product/` (entity, dto, errors, validator, repository interface & postgres adapter, usecase create & find, unit test, handler, and Wire provider).

---

# Cross-Feature Communication

In a microservice context, each service has a clear bounded context. However, within a single service, multiple features may need to interact.

## Principles

* Features **must NOT** import other feature packages directly.
* Communication between features is performed via **Interfaces injected via DI (Wire)**.
* If communication between features becomes too complex, that is an indicator that the features should be split into separate microservices.

## Pattern: Interface Injection

Feature `Order` requires `User` details:

```go
// internal/order/repository/interface.go
type UserReader interface {
    FindByID(ctx context.Context, id uint) (*entity.User, error)
}
```

```go
// internal/order/usecase/create.go
type CreateOrderUsecase struct {
    repo       Repository
    userReader UserReader
}
```

Wire injects the `UserReader` implementation provided by an adapter or exported provider from the user package.

## Pattern: Event-Driven (Async)

For asynchronous communication where no immediate response is needed:

```go
// internal/order/repository/interface.go
type EventPublisher interface {
    Publish(ctx context.Context, event Event) error
}
```

Order publishes an event; other features (or external services) subscribe and react independently.

## When to Split a Service

* Two features frequently communicate in a complex, synchronous manner.
* Features possess distinct deployment lifecycles.
* Features require independent horizontal scaling.
* The bounded context is no longer cohesive within a single service.

---

# Error Handling

## Principles

* Error codes are defined as **application-level codes** in `pkg/errorutil`, not raw gRPC codes.
* Each feature defines domain errors using `AppError` from `pkg/errorutil`.
* Infrastructure errors are wrapped into domain errors inside the repository adapter.
* Mapping from application codes to transport codes (gRPC/HTTP) is performed **once** in the interceptor, not in individual handlers.

## Application Error Codes

Define transport-agnostic error codes in `pkg/errorutil`:

```go
// pkg/errorutil/code.go
package errorutil

type Code int

const (
    CodeInternal      Code = iota // Unexpected error
    CodeNotFound                  // Resource not found
    CodeAlreadyExists             // Resource already exists
    CodeInvalidInput              // Validation failed
    CodeUnauthorized              // Authentication required
    CodeForbidden                 // Permission denied
    CodeConflict                  // State conflict
    CodePrecondition              // Precondition failed
)
```

## AppError Struct

```go
// pkg/errorutil/error.go
package errorutil

import "fmt"

type AppError struct {
    Code    Code
    Message string
    Err     error
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Err)
    }
    return e.Message
}

func (e *AppError) Unwrap() error {
    return e.Err
}

// New creates a new AppError
func New(code Code, message string) *AppError {
    return &AppError{Code: code, Message: message}
}

// Wrap creates an AppError wrapping an underlying error
func Wrap(code Code, message string, err error) *AppError {
    return &AppError{Code: code, Message: message, Err: err}
}
```

## Domain Errors per Feature

Each feature defines errors using `AppError`:

```go
// internal/user/errors/errors.go
package errors

import "go-feature-based-boilerplate/pkg/errorutil"

var (
    ErrNotFound      = errorutil.New(errorutil.CodeNotFound, "user not found")
    ErrAlreadyExists = errorutil.New(errorutil.CodeAlreadyExists, "user already exists")
    ErrInvalidInput  = errorutil.New(errorutil.CodeInvalidInput, "invalid input")
)
```

## Detailed Validation Errors

```go
// pkg/errorutil/validation.go
package errorutil

import "fmt"

type FieldError struct {
    Field   string
    Message string
}

func NewValidation(field, message string) *AppError {
    return &AppError{
        Code:    CodeInvalidInput,
        Message: fmt.Sprintf("validation failed on field %s: %s", field, message),
    }
}
```

## Error Wrapping in Infrastructure

```go
// internal/user/repository/postgres/repository.go
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
    var user entity.User
    if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, userErrors.ErrNotFound
        }
        return nil, errorutil.Wrap(errorutil.CodeInternal, "find user by id", err)
    }
    return &user, nil
}
```

## Error Interceptor (Centralized Mapping)

Mapping from application codes to gRPC status codes occurs **once** in the interceptor:

```go
// infrastructure/middleware/error_interceptor.go
package middleware

import (
    "errors"

    "go-feature-based-boilerplate/pkg/errorutil"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

func ErrorInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
        resp, err := handler(ctx, req)
        if err == nil {
            return resp, nil
        }

        var appErr *errorutil.AppError
        if errors.As(err, &appErr) {
            return nil, status.Error(toGRPCCode(appErr.Code), appErr.Message)
        }

        // Unexpected error — do not expose internal details to client
        return nil, status.Error(codes.Internal, "internal server error")
    }
}

func toGRPCCode(code errorutil.Code) codes.Code {
    switch code {
    case errorutil.CodeNotFound:
        return codes.NotFound
    case errorutil.CodeAlreadyExists:
        return codes.AlreadyExists
    case errorutil.CodeInvalidInput:
        return codes.InvalidArgument
    case errorutil.CodeUnauthorized:
        return codes.Unauthenticated
    case errorutil.CodeForbidden:
        return codes.PermissionDenied
    case errorutil.CodeConflict:
        return codes.AlreadyExists
    case errorutil.CodePrecondition:
        return codes.FailedPrecondition
    default:
        return codes.Internal
    }
}
```

## Clean Handlers

Because interceptors manage error translation, handlers simply return errors directly:

```go
// internal/user/handler/handler.go
func (h *Handler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    user, err := h.usecase.FindByID(ctx, uint(req.GetId()))
    if err != nil {
        return nil, err // interceptor translates to gRPC status
    }
    return toProtoResponse(user), nil
}
```

## gRPC to HTTP Mapping (Automatic via grpc-gateway)

`grpc-gateway` automatically translates gRPC status codes to corresponding HTTP status codes:

| App Code            | gRPC Code          | HTTP Status |
| ------------------- | ------------------ | ----------- |
| `CodeNotFound`      | `NotFound`         | 404         |
| `CodeAlreadyExists` | `AlreadyExists`    | 409         |
| `CodeInvalidInput`  | `InvalidArgument`  | 400         |
| `CodeInternal`      | `Internal`         | 500         |
| `CodeUnauthorized`  | `Unauthenticated`  | 401         |
| `CodeForbidden`     | `PermissionDenied` | 403         |

## Rationale for This Approach

| Aspect | Benefit |
| --- | --- |
| **Consistency** | All features use unified `errorutil.Code` with zero conflicting interpretations. |
| **Transport-Agnostic** | Domain errors know nothing of gRPC/HTTP, depending solely on `errorutil.Code`. |
| **Enforced Codes** | Errors cannot omit codes (`New` constructor enforces code requirement). |
| **Minimal Handlers** | Handlers need zero mapping logic; they return errors directly. |
| **Single Point of Change** | Mapping modifications occur solely in the interceptor. |

---

# Context & Timeout

## Principles

* `context.Context` must be propagated from handler through to the infrastructure layer.
* Individual layers **must NOT** create fresh root contexts (except for explicitly detached background tasks).
* Timeouts are enforced at the transport/middleware boundary, not inside usecases.

## Flow

```text
Handler (enforce timeout) → Usecase (propagate) → Repository (propagate) → Database (use context)
```

## Example

```go
// internal/user/handler/handler.go
func (h *Handler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    user, err := h.usecase.FindByID(ctx, req.GetId())
    if err != nil {
        return nil, err
    }
    return toProtoResponse(user), nil
}
```

```go
// internal/user/usecase/find.go
func (u *FindUsecase) FindByID(ctx context.Context, id uint) (*entity.User, error) {
    // Propagate context without creating fresh contexts
    return u.repo.FindByID(ctx, id)
}
```

```go
// internal/user/repository/postgres/repository.go
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
    var user entity.User
    // Context is utilized by GORM for timeout and cancellation
    if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
        // ...
    }
    return &user, nil
}
```

## Timeout Configuration

Set server default timeout in gRPC interceptor chain:

```go
// infrastructure/middleware/timeout_interceptor.go
func TimeoutInterceptor(defaultTimeout time.Duration) grpc.UnaryServerInterceptor
```

Configured in `bootstrap/grpc.go` using `cfg.Server.DefaultTimeout`.

---

# API Transport (gRPC + grpc-gateway)

## Principles

REST APIs are automatically derived from Protocol Buffer definitions using **grpc-gateway**. There is no need to write manual HTTP router handlers for core RPC methods.

## Proto with Gateway Annotations

```protobuf
syntax = "proto3";

package user.v1;

import "google/api/annotations.proto";

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = {
      get: "/api/v1/users/{id}"
    };
  }

  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
    option (google.api.http) = {
      post: "/api/v1/users"
      body: "*"
    };
  }
}
```

## Generated Output

From a single proto file, `protoc` generates:

| File | Function |
| --- | --- |
| `user.pb.go` | Strongly-typed message types |
| `user_grpc.pb.go` | gRPC server and client interfaces |
| `user.pb.gw.go` | HTTP reverse proxy gateway handlers |

## Bootstrap Gateway

```go
// bootstrap/gateway.go
func NewHTTPGateway(ctx context.Context, cfg *config.Config) (*http.Server, error)
```

Registers gRPC endpoints, mounts Prometheus `/metrics`, Swagger UI `/swagger/`, and wraps handlers with standard response envelopes and OpenTelemetry ingress tracing (`otelhttp`).

## Versioning

Versioning is built-in via proto package namespaces and URL paths:

```text
/api/v1/users
/api/v2/users
```

## Standard Base Response (Dual Approach: Gateway Envelope)

To balance high-performance serialization with client consistency:

1. **Internal gRPC (Service-to-Service)**:
   - Uses pure, strongly-typed Protobuf messages (`GetUserResponse`, `LoginResponse`).
   - Does NOT wrap payloads with generic wrappers like `google.protobuf.Any` at the proto level, preserving binary efficiency, strict typing, and backward compatibility.

2. **HTTP Gateway / REST (Client-to-Service)**:
   - All REST responses are automatically formatted into a uniform **Envelope Format** at the Gateway layer (`bootstrap/gateway.go`).
   - Uses `"status": "success"` for HTTP 2xx and `"status": "failed"` for HTTP 4xx/5xx.

### Success Response Format (HTTP 2xx)

```json
{
  "status": "success",
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "email": "user@example.com",
    "name": "John Doe"
  },
  "meta": {
    "request_id": "8f31b09b-6d33-4df4-8d48-6933bbec804a",
    "timestamp": "2026-09-04T08:52:07Z"
  }
}
```

### Error Response Format (HTTP 4xx / 5xx)

```json
{
  "status": "failed",
  "code": 400,
  "message": "validation failed on field email: invalid email format",
  "errors": null,
  "meta": {
    "request_id": "8f31b09b-6d33-4df4-8d48-6933bbec804a",
    "timestamp": "2026-09-04T08:52:07Z"
  }
}
```

### Gateway Envelope Characteristics:
- **`request_id` Correlation**: Gateway extracts `X-Request-ID` (or generates a fresh UUID if absent) and propagates it to response headers and gRPC metadata context (`x-request-id`).
- **Exemptions**: Non-API endpoints such as `/metrics` (Prometheus) and `/swagger/` (Swagger UI) bypass envelope wrapping to preserve tooling compatibility.

---

# Repository Pattern

Repositories are declared solely as Interfaces within `internal/<feature>/repository/interface.go`:

```go
// internal/user/repository/interface.go
type Repository interface {
    FindByID(ctx context.Context, id uint) (*entity.User, error)
    FindByEmail(ctx context.Context, email string) (*entity.User, error)
    Create(ctx context.Context, user *entity.User) error
    Update(ctx context.Context, user *entity.User) error
    Delete(ctx context.Context, id uint) error
}
```

Concrete implementations live inside feature adapter directories (Feature-First Adapter):
```text
internal/user/repository/postgres/repository.go
```

---

# Database Transaction Pattern (Unit of Work)

In Clean Architecture, usecases frequently need to execute operations across multiple repositories atomically (ACID) — for example: creating an Order and debiting a Wallet balance.

### Challenge & Clean Architecture Principles
Database driver objects like `*gorm.DB` or `*sql.Tx` **must not leak** into the domain usecase layer.

To resolve this, the boilerplate provides a `transaction.Manager` abstraction in `pkg/transaction/manager.go`:

```go
// pkg/transaction/manager.go
package transaction

import "context"

type Manager interface {
    RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

### 1. Usage in Usecase (Domain Purity)
Usecases depend exclusively on the `transaction.Manager` interface:

```go
type CheckoutUsecase struct {
    orderRepo  orderRepository.Repository
    walletRepo walletRepository.Repository
    txManager  transaction.Manager // Injected via Google Wire
}

func (u *CheckoutUsecase) Execute(ctx context.Context, req dto.CheckoutRequest) error {
    return u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
        // 1. Save order (using txCtx)
        if err := u.orderRepo.Create(txCtx, order); err != nil {
            return err // Triggers automatic rollback
        }

        // 2. Deduct wallet balance (using same txCtx)
        if err := u.walletRepo.Deduct(txCtx, userID, amount); err != nil {
            return err // Triggers automatic rollback
        }

        return nil // Commits transaction on nil error
    })
}
```

### 2. Infrastructure Implementation (`infrastructure/database/postgres`)
The concrete implementation in `infrastructure/database/postgres/tx_manager.go` wraps GORM transactions and injects the active transactional `*gorm.DB` into the context (`txCtx`).

Repositories automatically detect active transactions using `postgres.GetDB(ctx, r.db)`:

```go
// internal/order/repository/postgres/repository.go
func (r *OrderRepository) getDB(ctx context.Context) *gorm.DB {
    return postgres.GetDB(ctx, r.db)
}

func (r *OrderRepository) Create(ctx context.Context, item *entity.Order) error {
    return r.getDB(ctx).Create(item).Error
}
```

### Advantages:
1. **Zero Domain Leakage**: Usecase never imports GORM or raw SQL drivers.
2. **Backward-Compatible**: Repositories operate seamlessly inside or outside transactions without caller changes.
3. **Safe Nested Transactions**: Handles nested calls gracefully without duplicate root transactions.
4. **Unit-Test Friendly**: `transaction.Manager` is trivially mocked (`fn(ctx)`).

---

# External Services (Outbound Adapters)

Integrations with external services (via **gRPC** or **HTTP REST**) act as **Driven Adapters (Outbound / Secondary Adapters)**.

Two placement patterns apply:

1. **Shared / Cross-Cutting Services (Multi-Feature)**: Placed in `infrastructure/client/<service-name>/` (e.g. *Notification Service*, *Mailer/SMS*, *Audit Log*, *S3 Storage*).
2. **Domain-Specific Services (Exclusive to 1 Feature)**: Placed in `internal/<feature>/gateway/` or `client/` (e.g. *Payment Gateway Xendit* for Order/Billing).

---

## Example 1: Shared Service (`Notification Service` via gRPC)

Modules `User` (on signup) and `Order` (on checkout) both send notifications via remote `notification-service`.

### 1. Directory Structure
```text
infrastructure/
└── client/
    └── notification/           <- Public Client Implementation in Infrastructure
        ├── client.go           <- gRPC stub caller
        └── provider.go         <- Wire Provider for connection setup

internal/
├── user/
│   └── usecase/
│       ├── create.go
│       └── notifier.go         <- Consumer Port Interface
└── order/
    └── usecase/
        ├── checkout.go
        └── notifier.go         <- Consumer Port Interface
```

### 2. Client Implementation in `infrastructure/client/notification/client.go`
```go
package notification

import (
    "context"
    "fmt"

    notifpb "go-feature-based-boilerplate/gen/pb/notification"
    "google.golang.org/grpc"
)

type Client struct {
    grpcClient notifpb.NotificationServiceClient
}

func NewClient(conn *grpc.ClientConn) *Client {
    return &Client{grpcClient: notifpb.NewNotificationServiceClient(conn)}
}

func (c *Client) SendEmail(ctx context.Context, to string, subject string, body string) error {
    _, err := c.grpcClient.SendEmail(ctx, &notifpb.SendEmailRequest{
        To:      to,
        Subject: subject,
        Body:    body,
    })
    if err != nil {
        return fmt.Errorf("notification client: send email: %w", err)
    }
    return nil
}
```

### 3. Port Definition in Usecase (`internal/user/usecase/notifier.go`)
```go
package usecase

import "context"

type Notifier interface {
    SendEmail(ctx context.Context, to string, subject string, body string) error
}
```

---

## Decision Matrix

| Criterion | Shared Service | Domain-Specific Service |
| --- | --- | --- |
| **Location** | `infrastructure/client/<name>/` | `internal/<feature>/gateway/` |
| **Consumers** | Used by >= 2 different features | Relevant to 1 business domain only |
| **Real Example** | Notification, S3 Storage, Audit Log | Payment Gateway, Courier Tracking |
| **Usecase Relation** | Usecase declares consumer interface; Wire binds implementation | Usecase declares consumer interface; feature adapter implements it |

---

# Shared Packages (`pkg`)

## Principles

`pkg/` contains shared utilities, helpers, and generic wrappers that:

* Are utilized by **multiple features** or **multiple layers**.
* Are **generic** — containing zero domain business logic.
* Are **stateless** and self-contained.
* Can be extracted into standalone Go modules if needed by external projects.

## Structure

```text
pkg/
├── pagination/
│   ├── pagination.go
│   ├── cursor.go
│   ├── sort.go
│   └── gorm.go
├── errorutil/
│   ├── code.go
│   ├── error.go
│   └── validation.go
├── jwtutil/
│   └── jwt.go
├── transaction/
│   └── manager.go
├── contextutil/
│   └── context.go
├── routine/
│   ├── routine.go
│   ├── group.go
│   ├── pool.go
│   ├── retry.go
│   ├── singleflight.go
│   └── metrics.go
├── idempotency/
│   ├── idempotency.go
│   ├── executor.go
│   ├── memory.go
│   └── redis.go
├── pointer/
│   └── pointer.go
├── timeutil/
│   └── timeutil.go
├── stringutil/
│   └── stringutil.go
├── validator/
│   └── validator.go
├── httpclient/
│   ├── client.go
│   ├── options.go
│   └── transport.go
├── logger/
│   ├── logger.go
│   ├── zap.go
│   └── factory.go
└── cryptoutil/
    └── hash.go
```

### idempotency (Usecase-Driven Idempotency Pattern)

Guarantees critical state-mutation operations (payments, order creations, transfers) execute **exactly once** despite client retries or network drops.

* **Usecase-Driven (Explicit)** via **Go Generics (`Executor[T any]`)**.
* **Zero Global Overhead**: Read endpoints (`GET`) are untouched.
* **Dual Storage Engine**: `MemoryStorage` (testing) and `RedisStorage` (atomic `SetNX` for distributed clusters).

### routine (Concurrency & Goroutine Lifecycle Toolkit)

Provides a complete microservice concurrency toolkit:
1. **Lifecycle & Panic Recovery**: Safe execution preventing unhandled panics, context propagation (`WithoutCancel`), and graceful shutdown tracking (`WaitForShutdown`).
2. **Fan-Out / Fan-In (`Group`)**: Parallel task execution with semaphore concurrency limits (`WithLimit`) and overall timeouts (`WithTimeout`).
3. **Worker Pool (`Pool`)**: Bounded task queues with backpressure strategies (`StrategyBlock` and `StrategyDiscard`).
4. **Retry Engine (`Retry`)**: Synchronous and asynchronous retry operations with exponential backoff, randomized jitter, and custom error predicates (`RetryIf`).
5. **Generic Deduplicator / Singleflight (`Singleflight[T]`)**: Eliminates *Cache Stampede / Thundering Herd* by coalescing concurrent in-flight requests with identical keys into a single execution.
6. **Prometheus Metrics**: Exports real-time metrics for active goroutines, completed tasks, caught panics, and execution duration histograms at `/metrics`.

### httpclient (Resilient Outbound HTTP Client)

Enterprise outbound HTTP client:
1. **Automatic W3C Tracing**: Automatically injects `traceparent` headers via OpenTelemetry.
2. **`X-Request-ID` Correlation**: Propagates request correlation IDs automatically.
3. **Safe Connection Pooling**: Configured with pooled idle connections to prevent socket exhaustion.
4. **Transient Failure Retries**: Exponential backoff with jitter for HTTP 502/503/504 and network timeouts.
5. **HTTP QUERY Method (IETF RFC 10008)**: Official support for the safe, idempotent `QUERY` method allowing complex request bodies without URI length constraints.
6. **JSON Convenience Methods**: `GetJSON`, `PostJSON`, and `QueryJSON`.

---

# Testing

## Unit Tests
Co-located directly with source code:
```text
usecase/
├── create.go
└── create_test.go
```

## Integration Tests
```text
tests/integration/
```
Tests repositories and adapters against real containerized databases (Docker / Testcontainers).

## End-to-End Tests
```text
tests/e2e/
```
Tests full request/response lifecycles from gateway ingress to storage.

---

# Google Wire

Dependency injection is managed at compile time via Google Wire.

```go
// bootstrap/wire.go
//go:build wireinject

package bootstrap

import (
    "context"
    infraConfig "go-feature-based-boilerplate/infrastructure/config"
    "go-feature-based-boilerplate/infrastructure/database/postgres"
    "go-feature-based-boilerplate/infrastructure/telemetry"
    "go-feature-based-boilerplate/internal/auth"
    "go-feature-based-boilerplate/internal/health"
    "go-feature-based-boilerplate/internal/user"
    "github.com/google/wire"
)

var serverSet = wire.NewSet(
    NewGRPCServer,
    NewHTTPGateway,
    NewApp,
)

func InitializeApp(ctx context.Context) (*App, func(), error) {
    wire.Build(
        infraConfig.ProviderSet,
        ProvideLogger,
        postgres.ProviderSet,
        telemetry.ProviderSet,
        user.ProviderSet,
        auth.ProviderSet,
        health.ProviderSet,
        serverSet,
    )
    return nil, nil, nil
}
```

---

# Observability (Full-Stack APM)

## Principles
Observability is integrated across all communication boundaries and injected as cross-cutting middleware. Business logic does not invoke tracers or loggers directly except for domain audits.

## Structured JSON Logging (`pkg/logger`)
* **100% JSON Format**: Structured output with ISO8601 UTC timestamps, uppercase levels, and caller lines.
* **Zero Variadic Parameters**: Correlation fields (`trace_id`, `span_id`, `request_id`, `user_id`, `rpc.*`) are extracted automatically from `context.Context`.
* **Audit Payload Policy**:
  - `request`: Always logged (with sensitive PII data masked).
  - `response`: **Logged ONLY when an error occurs (`code != codes.OK`)** to maximize storage and CPU efficiency in production.

## Full-Stack Distributed Tracing Waterfall

```text
[HTTP Client / Browser]
        │
        ▼ (W3C traceparent header)
[HTTP Gateway :8080] ────── otelhttp.NewHandler (Ingress Root Span)
        │
        ▼ (gRPC Metadata Propagation)
[gRPC Server :50051] ────── otelgrpc.NewServerHandler (Server Span)
        │
        ├──► [GORM PostgreSQL] ── gorm.io/plugin/opentelemetry (DB Child Span + SQL Query)
        ├──► [Redis Cache]     ── redisotel.InstrumentTracing (Cache Child Span + Command)
        └──► [External HTTP]   ── pkg/httpclient (W3C Injected Outbound Span via RFC 10008)
```

1. **HTTP Ingress Tracing (`otelhttp`)**: Gateway REST (`bootstrap/gateway.go`) wraps the router with `otelhttp.NewHandler`, extracting W3C `traceparent` or initializing new root spans.
2. **gRPC Transport Tracing (`otelgrpc`)**: gRPC Server (`bootstrap/grpc.go`) tracks RPC spans and propagates context downstream.
3. **Database Query Tracing (`gorm.io/plugin/opentelemetry`)**: GORM (`infrastructure/database/postgres/db.go`) traces SQL execution durations and queries as child spans.
4. **Cache Command Tracing (`redisotel`)**: Redis (`infrastructure/cache/redis/redis.go`) traces Redis commands as child spans.
5. **Outbound HTTP Tracing (`pkg/httpclient`)**: Automatically injects `traceparent` headers into outgoing requests.
6. **Continuous Profiling (`net/http/pprof`)**: Mounted at `/debug/pprof/` in non-production environments to diagnose CPU, memory allocations, goroutine leaks, and mutex contention.

## Middleware Chain
```go
server := grpc.NewServer(
    grpc.StatsHandler(otelgrpc.NewServerHandler()),
    grpc.ChainUnaryInterceptor(
        middleware.RecoveryInterceptor(logger),                  // 1. Outermost panic recovery
        middleware.LoggingInterceptor(logger),                   // 2. Access logs & context enrichment
        middleware.TimeoutInterceptor(cfg.Server.DefaultTimeout), // 3. Request deadline enforcement
        middleware.AuthInterceptor(cfg),                         // 4. JWT token authentication
        middleware.ErrorInterceptor(),                           // 5. Domain error to gRPC status mapping
    ),
)
```

---

# Health Check & Graceful Shutdown

## Health Check
Implements standard gRPC Health Checking Protocol (`health.v1.HealthService`) along with HTTP gateway readiness and liveness probes (`/healthz`, `/readyz`).

## Graceful Shutdown
```go
func (a *App) Shutdown(ctx context.Context) {
    // 1. Drain HTTP gateway ingress
    a.httpServer.Shutdown(ctx)

    // 2. Stop gRPC server gracefully
    a.grpcServer.GracefulStop()

    // 3. Wait for all active background goroutines to finish
    routine.WaitForShutdown(ctx)

    // 4. Close database connection pools
    if sqlDB, err := a.db.DB(); err == nil {
        _ = sqlDB.Close()
    }

    // 5. Flush telemetry pipelines
    a.tracerProvider.Shutdown(ctx)
    a.meterProvider.Shutdown(ctx)

    // 6. Flush logger buffers
    _ = a.logger.Sync()
}
```

---

# Adding a New Feature (Checklist)

1. Create feature directory skeleton in `internal/<feature>/`.
2. Declare entities (`entity/`) and repository interfaces (`repository/interface.go`).
3. Implement usecase workflows (`usecase/`).
4. Define Protocol Buffer contracts (`api/proto/<feature>/v1/<feature>.proto`) with grpc-gateway HTTP annotations.
5. Compile proto files using `make protogen`.
6. Implement gRPC handlers (`handler/handler.go`).
7. Implement database repository adapters (`repository/postgres/repository.go`).
8. Create Wire provider set (`provider.go`).
9. Register in `bootstrap/wire.go`, `bootstrap/grpc.go`, and `bootstrap/gateway.go`.
10. Create database migrations if schema changes are required (`make migration name=...`).
11. Run tests with race detection: `go test -race ./...`.
12. Build binary: `make build`.

---

# Design Philosophy

* Feature-First Modularity
* Clean Architecture Domain Purity
* Hexagonal Ports & Adapters
* Technology & Framework Independence
* Strict Dependency Inversion
* Microservice Scalability
* 100% Observable (Full-Stack APM)
* High-Performance Dual-Protocol Transport (gRPC + REST Gateway)

---

# Summary

| Directory | Purpose |
| --- | --- |
| `cmd` | Minimal Application Entry Point |
| `bootstrap` | Composition Root + Server Setup & Lifecycle |
| `internal` | Feature-First Business Logic + Domain Interfaces |
| `infrastructure` | Framework Drivers, Databases, Telemetry, and Adapters |
| `pkg` | Generic, Reusable, and Agnostic Utility Libraries |
| `api` | API Contracts (Protobuf + HTTP Gateway Annotations) |
| `gen` | Machine-Generated Artifacts (pb, openapi, mocks) |
| `configs` | Application Configuration Files |
| `migrations` | Managed SQL Database Migrations |
| `tests` | Integration and End-to-End Test Suites |

This blueprint provides an enterprise-grade Go microservice foundation designed to scale from small services to mission-critical systems without structural refactoring.\n