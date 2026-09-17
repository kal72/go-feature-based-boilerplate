# Go Microservice Blueprint (Feature-Based Modular Clean Architecture)

Feature-Based Modular Clean Architecture dengan prinsip Hexagonal Architecture (Ports & Adapters) dan pendekatan Lightweight Domain-Driven Design (DDD), yang memisahkan business logic dari infrastructure sehingga aplikasi tetap modular, testable, dan technology-agnostic.

> **Tujuan**
>
> Blueprint ini digunakan sebagai standar untuk seluruh microservice Go yang dibangun menggunakan:
>
> * Go
> * gRPC + grpc-gateway (REST via gateway)
> * Google Wire
> * GORM
> * PostgreSQL
> * MongoDB
> * Docker
> * JWT
> * Unit Test
> * Integration Test
> * Observability (Logging, Metrics, Tracing)

---

# Design Principles

## 1. Separation of Responsibility

Setiap folder hanya memiliki satu tanggung jawab.

| Folder           | Responsibility                                 |
| ---------------- | ---------------------------------------------- |
| `cmd`            | Application Entry Point                        |
| `bootstrap`      | Dependency Injection & Application Composition |
| `internal`       | Business Logic                                 |
| `infrastructure` | Framework & External Implementation            |
| `pkg`            | Shared Library, Helper, Wrapper (Generic)      |
| `api`            | API Contract (Proto + OpenAPI)                 |
| `gen`            | Generated Code                                 |
| `configs`        | Configuration Files (YAML, env)                |
| `migrations`     | Database Migration                             |
| `tests`          | Integration & E2E Test                         |

---

## 2. Clean Architecture

```
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

Business Logic tidak mengetahui implementasi:

* HTTP
* Database
* Redis
* Kafka
* gRPC
* External Service

Business hanya mengenal Interface.

---

## 3. Feature First

Seluruh business dikelompokkan berdasarkan feature.

Contoh:

```
User
Auth
Order
Payment
Inventory
```

Bukan berdasarkan layer global.

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
│   └── stringutil/
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
│   │
│   ├── logger/
│   │
│   ├── telemetry/
│   │
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

Hanya sebagai entry point.

Boleh import:

* bootstrap

Tidak boleh import:

* internal secara langsung

---

## bootstrap

Composition Root.

Berisi:

* Google Wire
* gRPC Server
* grpc-gateway (REST)
* Middleware Registration

Bootstrap boleh mengetahui seluruh project.

Karena ini microservice, satu service memiliki scope feature yang terbatas. File `bootstrap/grpc.go` dan `bootstrap/gateway.go` tidak akan membengkak secara signifikan. Jika service mulai memiliki terlalu banyak handler, itu sinyal bahwa service perlu dipecah (split microservice), bukan menambah complexity di routing layer.

---

## internal

Berisi seluruh Business Logic.

Tidak boleh bergantung kepada:

* GORM
* Mongo Driver
* Redis Client
* Kafka
* HTTP Client
* gRPC Client
* Framework apapun

Business hanya mengenal Interface.

---

## infrastructure

Berisi implementasi teknis.

Contoh:

* PostgreSQL
* MongoDB
* Redis
* Kafka
* S3
* SMTP
* HTTP Client
* gRPC Client

Tidak boleh berisi Business Rule.

---

## pkg

Berisi shared utility, helper, dan wrapper yang digunakan secara global oleh seluruh layer.

Boleh di-import oleh:

* internal
* infrastructure
* bootstrap

Tidak boleh bergantung kepada:

* internal (business logic)
* infrastructure (framework implementation)
* bootstrap

`pkg` harus **self-contained** dan **stateless**. Tidak boleh memiliki dependency ke layer manapun di atasnya.

---

## api

Berisi source API Contract.

Contoh:

* Proto (dengan grpc-gateway annotations)
* OpenAPI (generated dari proto)

Tidak boleh ada business logic.

---

## gen

Hanya hasil generate.

Tidak boleh diedit manual.

Termasuk:

* `*.pb.go` (protobuf)
* `*_grpc.pb.go` (gRPC service)
* `*.pb.gw.go` (grpc-gateway)
* Mock files (generated via mockgen/mockery)

---

## configs vs infrastructure/config

| Path                   | Isi                                              |
| ---------------------- | ------------------------------------------------ |
| `configs/`             | File konfigurasi (YAML, `.env`, JSON)            |
| `infrastructure/config/` | Code untuk loading, parsing, dan validasi config |

Contoh `infrastructure/config/`:

```go
package config

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    Logger   LoggerConfig
}

func Load(path string) (*Config, error) {
    // viper atau koanf loading logic
}
```

---

# Feature Structure

Contoh module User:

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

## Otomatisasi Pembuatan Fitur (`make new-feature`)

Untuk mempercepat pengembangan dan menjamin konsistensi struktur antar tim, boilerplate menyediakan generator otomatis via Makefile:

```bash
make new-feature name=product
```

Perintah ini akan mengeksekusi shell script [scripts/new-feature.sh](file:///Users/a2375/Projects/go-feature-based-boilerplate/scripts/new-feature.sh) dan secara otomatis menghasilkan:
- File Proto API contract di `api/proto/product/product.proto`.
- Skeleton lengkap folder `internal/product/` (entity, dto, errors, validator, repository interface & postgres adapter, usecase create & find, unit test, handler, dan provider Wire).

---

# Cross-Feature Communication

Dalam konteks microservice, satu service memiliki bounded context yang jelas. Namun dalam satu service, beberapa feature mungkin perlu saling berinteraksi.

## Prinsip

* Feature **tidak boleh** import package feature lain secara langsung.
* Komunikasi antar feature dilakukan melalui **Interface yang di-inject via DI (Wire)**.
* Jika kebutuhan komunikasi antar feature terlalu kompleks, itu sinyal bahwa feature tersebut harus menjadi service terpisah.

## Pattern: Interface Injection

Feature `Order` membutuhkan data `User`:

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

Wire akan meng-inject implementasi `UserReader` yang berasal dari infrastructure atau dari usecase user yang sudah ada.

## Pattern: Event-Driven (Async)

Untuk komunikasi yang tidak memerlukan response langsung:

```go
// internal/order/repository/interface.go
type EventPublisher interface {
    Publish(ctx context.Context, event Event) error
}
```

Order mempublish event, feature lain (atau service lain) yang subscribe akan bereaksi secara independen.

## Kapan Harus Split Service

* Dua feature sering berkomunikasi secara synchronous dan complex.
* Feature memiliki lifecycle deployment yang berbeda.
* Feature membutuhkan scaling yang berbeda.
* Bounded context sudah tidak relevan dalam satu service.

---

# Error Handling

## Prinsip

* Error code didefinisikan sebagai **application-level code** di `pkg/errorutil`, bukan gRPC code.
* Setiap feature mendefinisikan domain error menggunakan `AppError` dari `pkg/errorutil`.
* Error dari infrastructure di-wrap menjadi domain error di repository implementation.
* Mapping dari application code ke transport code (gRPC/HTTP) dilakukan **satu kali** di interceptor, bukan di setiap handler.

## Application Error Code

Definisikan error code di `pkg/errorutil` yang tidak bergantung pada transport apapun:

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

// New membuat AppError baru
func New(code Code, message string) *AppError {
    return &AppError{Code: code, Message: message}
}

// Wrap membuat AppError dengan underlying error
func Wrap(code Code, message string, err error) *AppError {
    return &AppError{Code: code, Message: message, Err: err}
}
```

## Domain Error per Feature

Setiap feature mendefinisikan error menggunakan `AppError`:

```go
// internal/user/errors/errors.go
package errors

import "my-service/pkg/errorutil"

var (
    ErrNotFound      = errorutil.New(errorutil.CodeNotFound, "user not found")
    ErrAlreadyExists = errorutil.New(errorutil.CodeAlreadyExists, "user already exists")
    ErrInvalidInput  = errorutil.New(errorutil.CodeInvalidInput, "invalid input")
)
```

## Validation Error dengan Detail

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

## Error Wrapping di Infrastructure

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

Mapping dari application code ke gRPC status dilakukan **satu kali** di interceptor:

```go
// infrastructure/middleware/error_interceptor.go
package middleware

import (
    "errors"

    "my-service/pkg/errorutil"
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

        // Unexpected error — jangan expose detail ke client
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

## Handler (Sangat Simple)

Karena interceptor yang handle mapping, handler cukup return error langsung:

```go
// internal/user/handler/handler.go
func (h *Handler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    user, err := h.usecase.FindByID(ctx, uint(req.GetId()))
    if err != nil {
        return nil, err // interceptor yang mapping ke gRPC status
    }
    return toProtoResponse(user), nil
}
```

## gRPC to HTTP Mapping (Otomatis via grpc-gateway)

grpc-gateway akan otomatis memetakan gRPC status codes ke HTTP status codes:

| App Code           | gRPC Code          | HTTP Status |
| ------------------ | ------------------ | ----------- |
| `CodeNotFound`     | `NotFound`         | 404         |
| `CodeAlreadyExists`| `AlreadyExists`    | 409         |
| `CodeInvalidInput` | `InvalidArgument`  | 400         |
| `CodeInternal`     | `Internal`         | 500         |
| `CodeUnauthorized` | `Unauthenticated`  | 401         |
| `CodeForbidden`    | `PermissionDenied` | 403         |

## Kenapa Pendekatan Ini

| Aspek | Benefit |
|-------|---------|
| Konsistensi | Semua tim pakai `errorutil.Code` yang sama, tidak ada interpretasi berbeda |
| Transport-agnostic | Domain error tidak kenal gRPC/HTTP, hanya kenal `errorutil.Code` |
| Tidak bisa lupa | Error tanpa code tidak mungkin dibuat (constructor `New` wajib pakai code) |
| Handler minimal | Handler tidak perlu logic mapping, cukup return error |
| Single point of change | Kalau mau ubah mapping, hanya ubah di interceptor |

---

# Context & Timeout

## Prinsip

* `context.Context` harus dipropagasikan dari handler sampai ke infrastructure layer.
* Setiap layer **tidak boleh** membuat context baru (kecuali untuk background job).
* Timeout di-set di level handler atau middleware, bukan di usecase.

## Flow

```
Handler (set timeout) → Usecase (propagate) → Repository (propagate) → Database (use context)
```

## Contoh

```go
// internal/user/handler/handler.go
func (h *Handler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    // Context sudah memiliki timeout dari gRPC server config atau interceptor
    user, err := h.usecase.FindByID(ctx, req.GetId())
    if err != nil {
        return nil, status.Error(mapErrorToStatus(err), err.Error())
    }
    return toProtoResponse(user), nil
}
```

```go
// internal/user/usecase/find.go
func (u *FindUsecase) FindByID(ctx context.Context, id uint) (*entity.User, error) {
    // Propagate context, jangan buat context baru
    return u.repo.FindByID(ctx, id)
}
```

```go
// internal/user/repository/postgres/repository.go
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
    var user entity.User
    // Context digunakan oleh GORM untuk timeout dan cancellation
    if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
        // ...
    }
    return &user, nil
}
```

## Timeout Configuration

Set default timeout di gRPC server level:

```go
// bootstrap/grpc.go
server := grpc.NewServer(
    grpc.UnaryInterceptor(
        grpc_middleware.ChainUnaryServer(
            grpc_ctxtags.UnaryServerInterceptor(),
            grpc.UnaryServerInterceptor(timeoutInterceptor(30 * time.Second)),
        ),
    ),
)
```

---

# API Transport (gRPC + grpc-gateway)

## Prinsip

REST API dihasilkan secara otomatis dari proto definition menggunakan **grpc-gateway**. Tidak perlu menulis handler REST secara manual.

## Proto dengan Gateway Annotations

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

Dari satu proto file, dihasilkan:

| File              | Fungsi                          |
| ----------------- | ------------------------------- |
| `user.pb.go`      | Message types                   |
| `user_grpc.pb.go` | gRPC server/client interface    |
| `user.pb.gw.go`   | REST gateway handler (auto)     |

## Bootstrap Gateway

```go
// bootstrap/gateway.go
func NewGateway(ctx context.Context, grpcAddr string) (*runtime.ServeMux, error) {
    mux := runtime.NewServeMux()
    opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

    if err := pb.RegisterUserServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
        return nil, err
    }
    if err := pb.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
        return nil, err
    }

    return mux, nil
}
```

## Versioning

Versioning sudah built-in via proto package dan URL path:

```
/api/v1/users
/api/v2/users
```

Didefinisikan langsung di proto annotations.

## Standard Base Response (Pendekatan Dual: Gateway Envelope)

Untuk menjaga performa dan keseragaman antar channel komunikasi, arsitektur ini menerapkan **Pendekatan Dual**:

1. **Internal gRPC (Service-to-Service)**:
   - Tetap menggunakan message Protobuf murni yang *strongly-typed* (misalnya `GetUserResponse`, `LoginResponse`).
   - Tidak membungkus payload dengan generic wrapper seperti `google.protobuf.Any` atau `BaseResponse` di level proto, guna menjaga efisiensi serialisasi binary, type-safety, dan backward compatibility.

2. **HTTP Gateway / REST (Client-to-Service)**:
   - Semua respons REST di-wrap secara otomatis menjadi **Envelope Format** seragam di lapisan Gateway (`bootstrap/gateway.go`).
   - Menggunakan field `"status": "success"` untuk HTTP 2xx dan `"status": "failed"` untuk HTTP 4xx / 5xx.

### Format Respons Sukses (HTTP 2xx)

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

### Format Respons Error (HTTP 4xx / 5xx)

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

### Karakteristik Gateway Envelope:
- **`request_id` Correlation**: Gateway mengekstrak header `X-Request-ID` (atau otomatis men-generate UUID baru jika absen) dan mempropagasi nilainya ke header HTTP respons serta gRPC metadata context (`x-request-id`), sehingga logging gRPC dan HTTP response memiliki ID korelasi yang identik.
- **Exemptions**: Endpoint non-API seperti `/metrics` (Prometheus Scraper) dan `/swagger/` (Swagger UI) dilewati tanpa dibungkus envelope agar kompatibilitas tooling tetap terjaga.

---

# Repository Pattern

Repository hanya berupa Interface di dalam `internal`.

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

Implementasi berada di dalam folder fitur (Feature-First Adapter):
```text
internal/user/repository/postgres/repository.go
```

---

# Database Transaction Pattern (Unit of Work)

Dalam Clean Architecture, usecase sering kali perlu mengeksekusi operasi ke beberapa repository sekaligus secara atomik (ACID) — misalnya: membuat Order dan memotong saldo Wallet. 

### Tantangan & Prinsip Clean Architecture
Objek database driver seperti `*gorm.DB` atau `*sql.Tx` **tidak boleh bocor** ke domain usecase layer.

Untuk mengatasi ini, boilerplate menyediakan abstraksi `transaction.Manager` di `pkg/transaction/manager.go`:

```go
// pkg/transaction/manager.go
package transaction

import "context"

type Manager interface {
    RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

### 1. Penggunaan di Usecase (Domain Purity)
Usecase hanya bergantung pada interface `transaction.Manager` tanpa mengetahui implementasi database:

```go
type CheckoutUsecase struct {
    orderRepo  orderRepository.Repository
    walletRepo walletRepository.Repository
    txManager  transaction.Manager // Di-inject via Google Wire
}

func (u *CheckoutUsecase) Execute(ctx context.Context, req dto.CheckoutRequest) error {
    return u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
        // 1. Simpan order (menggunakan txCtx)
        if err := u.orderRepo.Create(txCtx, order); err != nil {
            return err // Otomatis rollback
        }

        // 2. Potong saldo wallet (menggunakan txCtx yang sama)
        if err := u.walletRepo.Deduct(txCtx, userID, amount); err != nil {
            return err // Otomatis rollback
        }

        return nil // Otomatis commit jika sukses
    })
}
```

### 2. Implementasi di Infrastructure (`infrastructure/database/postgres`)
Implementasi konkret [infrastructure/database/postgres/tx_manager.go](file:///Users/a2375/Projects/go-feature-based-boilerplate/infrastructure/database/postgres/tx_manager.go) membungkus transaksi GORM dan menyimpan instance `*gorm.DB` transaksi ke dalam context (`txCtx`).

Repository di setiap fitur secara otomatis mendeteksi apakah context sedang berada dalam transaksi menggunakan helper `postgres.GetDB`:

```go
// internal/order/repository/postgres/repository.go
func (r *OrderRepository) getDB(ctx context.Context) *gorm.DB {
    return postgres.GetDB(ctx, r.db)
}

func (r *OrderRepository) Create(ctx context.Context, item *entity.Order) error {
    // Jika ctx membawa transaksi aktif dari RunInTransaction, query otomatis
    // dieksekusi di dalam transaksi tersebut. Jika tidak, menggunakan default db pool.
    return r.getDB(ctx).Create(item).Error
}
```

### Keunggulan Desain Ini:
1. **Zero Domain Leakage**: Usecase tidak pernah mengimpor GORM atau SQL driver.
2. **Backward Compatible**: Repository tetap dapat dipanggil secara mandiri di luar transaksi tanpa modifikasi kode pemanggil.
3. **Safe Nested Transactions**: Jika method di dalam `RunInTransaction` memanggil method lain yang juga membungkus `RunInTransaction`, sistem mendeteksi transaksi aktif di context dan tidak membuat root transaksi ganda.
4. **Unit-Test Friendly**: `transaction.Manager` sangat mudah di-mock dalam unit test usecase (cukup panggil `fn(ctx)`).

---

# External Service (Outbound Adapters)

Dalam arsitektur Clean & Hexagonal, integrasi dengan service eksternal (baik via **gRPC Client** maupun **HTTP REST Client**) dikategorikan sebagai **Driven Adapter (Outbound / Secondary Adapter)**.

Terdapat dua pola penempatan tergantung sifat penggunaannya:

1. **Shared / Cross-Cutting Services (Publik / Multi-Fitur)**: Diletakkan di `infrastructure/client/<service-name>/`. Digunakan ketika service tersebut dibutuhkan oleh banyak fitur (contoh: *Notification Service*, *Mailer/SMS*, *Audit Log*, *Storage S3/MinIO*, *Master User Service*).
2. **Domain-Specific Services (Eksklusif 1 Fitur)**: Diletakkan di `internal/<feature>/gateway/` atau `client/`. Digunakan ketika integrasi pihak ketiga tersebut hanya relevan dan hanya boleh diakses oleh satu domain bisnis saja (contoh: *Midtrans/Stripe Payment Gateway* untuk Order/Billing, *Kurir Tracking* untuk Shipping).

---

## Contoh Kasus 1: Shared Service (`Notification Service` via gRPC)

Kasus: Modul `User` (saat register) dan modul `Order` (saat checkout) sama-sama perlu mengirim email notifikasi melalui service gRPC eksternal `notification-service`.

### 1. Struktur Folder
```text
infrastructure/
└── client/
    └── notification/           <- Implementasi Client Publik di Infrastructure
        ├── client.go           <- Logika pemanggilan stub gRPC
        └── provider.go         <- Provider Wire untuk inisialisasi koneksi

internal/
├── user/
│   └── usecase/
│       ├── create.go
│       └── notifier.go         <- Port Interface (apa yang dibutuhkan fitur User)
└── order/
    └── usecase/
        ├── checkout.go
        └── notifier.go         <- Port Interface (apa yang dibutuhkan fitur Order)
```

### 2. Implementasi Client di `infrastructure/client/notification/client.go`
Client ini membungkus stub gRPC yang digenerasi dari proto service notifikasi:

```go
package notification

import (
    "context"
    "fmt"

    notifpb "go-feature-based-boilerplate/gen/pb/notification"
    "google.golang.org/grpc"
)

// Client mengelola panggilan gRPC ke remote notification-service
type Client struct {
    grpcClient notifpb.NotificationServiceClient
}

// NewClient membuat instance notification client dari koneksi gRPC
func NewClient(conn *grpc.ClientConn) *Client {
    return &Client{
        grpcClient: notifpb.NewNotificationServiceClient(conn),
    }
}

// SendEmail mengirim email melalui remote service
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

### 3. Koneksi gRPC di `infrastructure/client/notification/provider.go`
```go
package notification

import (
    "context"
    "fmt"
    "time"

    "go-feature-based-boilerplate/infrastructure/config"
    "github.com/google/wire"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

// ProviderSet mengekspor inisialisasi koneksi dan client
var ProviderSet = wire.NewSet(
    NewConnection,
    NewClient,
)

func NewConnection(cfg *config.Config) (*grpc.ClientConn, func(), error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    conn, err := grpc.DialContext(ctx, cfg.ExternalServices.NotificationGRPCAddr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithBlock(),
    )
    if err != nil {
        return nil, nil, fmt.Errorf("dial notification service: %w", err)
    }

    cleanup := func() {
        _ = conn.Close()
    }
    return conn, cleanup, nil
}
```

### 4. Definisi Port (Interface) di Usecase Fitur (`internal/user/usecase/notifier.go`)
Prinsip **Dependency Inversion**: Usecase tidak mengimpor package `infrastructure/client/notification` secara langsung, melainkan mendefinisikan port interface yang dibutuhkan:

```go
package usecase

import "context"

// Notifier adalah kontrak consumer yang dibutuhkan oleh domain User
type Notifier interface {
    SendEmail(ctx context.Context, to string, subject string, body string) error
}
```

Di usecase penggunaannya:
```go
// internal/user/usecase/create.go
type CreateUsecase struct {
    repo     repository.Repository
    notifier Notifier // <-- Mengonsumsi interface, bukan struct konkret
}

func (u *CreateUsecase) Execute(ctx context.Context, req dto.CreateUserRequest) (*entity.User, error) {
    // 1. Simpan user ke database
    user, err := u.repo.Create(ctx, ...)
    if err != nil {
        return nil, err
    }

    // 2. Kirim email sambutan via interface
    _ = u.notifier.SendEmail(ctx, user.Email, "Selamat Datang!", "Akun Anda berhasil dibuat.")

    return user, nil
}
```

### 5. Wiring via Google Wire
Karena `*notification.Client` memiliki method `SendEmail(ctx, to, subject, body)`, struct tersebut otomatis memenuhi interface `usecase.Notifier`. Di `internal/user/provider.go`:

```go
func ProvideNotifier(client *notification.Client) usecase.Notifier {
    return client
}

var ProviderSet = wire.NewSet(
    // ...
    ProvideNotifier,
)
```

---

## Contoh Kasus 2: Domain-Specific Service (`Payment Gateway` via REST/gRPC)

Jika service eksternal hanya digunakan oleh satu fitur dan tidak boleh diakses oleh fitur lain, letakkan adapter di dalam folder fitur tersebut:

```text
internal/order/
├── usecase/
│   ├── checkout.go
│   └── payment_gateway.go      <- Port Interface (kontrak domain)
└── gateway/ (atau client/)     <- Adapter Outbound khusus Order
    └── xendit/
        └── client.go           <- Implementasi HTTP/SDK Xendit khusus Order
```

### Port Interface di `internal/order/usecase/payment_gateway.go`:
```go
package usecase

import (
    "context"
    "go-feature-based-boilerplate/internal/order/entity"
)

type PaymentGateway interface {
    CreateInvoice(ctx context.Context, orderID string, amount float64) (*entity.Invoice, error)
    GetStatus(ctx context.Context, invoiceID string) (entity.PaymentStatus, error)
}
```

---

## Ringkasan Aturan Keputusan

| Kriteria | Shared Service | Domain-Specific Service |
| :--- | :--- | :--- |
| **Lokasi Kode Client** | `infrastructure/client/<name>/` | `internal/<feature>/gateway/` |
| **Cakupan Konsumen** | Digunakan oleh >= 2 fitur berbeda | Hanya relevan untuk 1 fitur domain |
| **Contoh Nyata** | Notification, S3 Storage, Audit Log, Event Broker | Payment Gateway (Xendit), Ekspedisi Pengiriman |
| **Hubungan ke Usecase** | Usecase mendefinisikan consumer interface lokal; implementasi di-inject via Wire | Usecase mendefinisikan interface lokal; adapter fitur mengimplementasikannya |

---

# Shared Package (pkg)

## Prinsip

`pkg/` adalah tempat untuk shared utility, helper, dan wrapper yang:

* Digunakan oleh **lebih dari satu feature** atau **lebih dari satu layer**.
* Bersifat **generic** — tidak mengandung business logic.
* **Stateless** — tidak menyimpan state atau dependency ke layer lain.
* Bisa di-extract menjadi Go module terpisah jika dibutuhkan oleh service lain.

## Struktur

```text
pkg/
├── pagination/
│   └── pagination.go
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
└── cryptoutil/
    └── hash.go
```

### jwtutil

Menyediakan utilitas terpusat untuk pembuatan dan validasi token JWT (RFC 7519 & HMAC-SHA256):

```go
// pkg/jwtutil/jwt.go
package jwtutil

type Claims struct {
    jwt.RegisteredClaims
    UserID uint   `json:"user_id,omitempty"`
    Role   string `json:"role,omitempty"`
}

// GenerateToken membuat dan menandatangani JWT access token baru
func GenerateToken(userID uint, role string, secret []byte, ttl time.Duration) (string, error)

// ValidateToken memverifikasi signature, masa aktif (exp), dan mengekstrak Claims
func ValidateToken(tokenStr string, secret []byte) (*Claims, error)
```

### idempotency (Usecase-Driven Idempotency Pattern)

Menyediakan jaminan bahwa operasi mutasi kritis (seperti pembayaran, pembuatan pesanan, transfer dana) dieksekusi **tepat satu kali** (*exactly-once semantics*) meskipun terjadi klik ganda dari klien atau *retry* otomatis dari jaringan.

Pendekatan yang digunakan adalah **Usecase-Driven (Explicit)** berbasis **Go Generics (`Executor[T any]`)**, sehingga:
- **Zero Global Overhead**: Endpoint baca (`GET`) dan mutasi non-kritis lainnya 100% tidak terbebani oleh pengecekan Redis.
- **Type-Safe**: Mengembalikan struct/pointer domain entitas secara langsung tanpa perlu casting manual `.(T)`.
- **Dual Storage Engine**: Didukung oleh `MemoryStorage` (zero-dependency, cocok untuk testing) dan `RedisStorage` (atomik `SetNX` untuk multi-instance cluster).

```go
// pkg/idempotency/idempotency.go
package idempotency

type Storage interface {
    Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)
    Set(ctx context.Context, key string, record *Record, ttl time.Duration) error
    Get(ctx context.Context, key string) (*Record, error)
    Delete(ctx context.Context, key string) error
}

// FromContext mengekstrak Idempotency-Key dari incoming gRPC metadata
func FromContext(ctx context.Context) string
```

#### Contoh Penggunaan di Layer Usecase:
```go
type PaymentUsecase struct {
    repo        PaymentRepository
    idempExec   *idempotency.Executor[*entity.Payment]
}

func NewPaymentUsecase(repo PaymentRepository, storage idempotency.Storage) *PaymentUsecase {
    return &PaymentUsecase{
        repo: repo,
        idempExec: idempotency.NewExecutor[*entity.Payment](storage,
            idempotency.WithLockTTL(2*time.Minute),      // Lock in-progress
            idempotency.WithCompleteTTL(24*time.Hour),   // Cache respons sukses
        ),
    }
}

func (u *PaymentUsecase) ProcessPayment(ctx context.Context, req dto.PaymentRequest) (*entity.Payment, error) {
    // Ambil idempotency key dari header HTTP "Idempotency-Key" / "X-Idempotency-Key"
    idempKey := idempotency.FromContext(ctx)

    // Bungkus operasi mutasi: jika key sama datang lagi, respons cache langsung dikembalikan!
    return u.idempExec.Execute(ctx, idempKey, func(execCtx context.Context) (*entity.Payment, error) {
        // 1. Potong saldo & simpan ke database
        payment, err := u.repo.CreatePayment(execCtx, req)
        if err != nil {
            return nil, err // Error otomatis membatalkan lock, sehingga klien bisa retry
        }
        return payment, nil
    })
}
```

### routine (Concurrency & Goroutine Lifecycle Helper)

Menyediakan concurrency toolkit lengkap untuk produksi microservice:
1. **Lifecycle & Panic Recovery**: Eksekusi aman terhadap uncaught panic, context propagation (`WithoutCancel`), dan graceful shutdown koordinasi (`WaitForShutdown`).
2. **Fan-Out / Fan-In (`Group`)**: Koordinasi multi-task paralel dengan semaphore rate-limiting (`WithLimit`), timeout deadline (`WithTimeout`), dan pembatalan saudara otomatis pada error.
3. **Worker Pool (`Pool`)**: Antrean tugas bounded dengan strategi backpressure (`StrategyBlock` dan `StrategyDiscard`), panic recovery per-worker, dan graceful drain saat `Stop()`.
4. **Retry Engine (`Retry`)**: Mekanisme retry sinkron maupun asinkron berulang dengan exponential backoff, jitter acak, dan filter error kustom (`RetryIf`).
5. **Generic Deduplicator / Singleflight (`Singleflight[T]`)**: Mencegah *Cache Stampede / Thundering Herd* dengan menggabungkan pemanggilan identik konkuren menjadi 1 eksekusi saja, sepenuhnya type-safe via Go Generics.
6. **Observability Prometheus (`/metrics`)**: Metrik real-time goroutine aktif, total tugas dieksekusi, counter panic tertangkap, serta histogram latensi durasi eksekusi.

```go
// pkg/routine/routine.go
package routine

// Go menjalankan fungsi di goroutine terpisah dengan proteksi panic recovery otomatis,
// pelacakan shutdown graceful, serta metrik observabilitas.
func Go(ctx context.Context, fn func(ctx context.Context))

// GoDetached menjalankan background task yang tidak terpengaruh oleh pembatalan klien/timeout request,
// tetapi SELURUH context values (trace_id, request_id, user_id, role) tetap terbawa.
func GoDetached(ctx context.Context, fn func(bgCtx context.Context))

// WaitForShutdown menunggu seluruh background task aktif selesai atau hingga ctx shutdown timeout.
func WaitForShutdown(ctx context.Context) error

// NewGroup menginisialisasi koordinator tugas konkuren (Fan-Out / Fan-In)
// dengan dukungan batas konkurensi (WithLimit) dan timeout menyeluruh (WithTimeout).
func NewGroup(ctx context.Context, opts ...Option) *Group

// NewPool menginisialisasi bounded worker pool dengan fixed worker goroutines dan backpressure.
func NewPool(workerCount int, opts ...PoolOption) *Pool

// Retry mengeksekusi operasi sinkron dengan exponential backoff dan jitter.
func Retry(ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) error) error

// GoWithRetry mengeksekusi fungsi secara asinkron di background goroutine dengan konfigurasi retry.
func GoWithRetry(ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) error)

// GoDetachedWithRetry mengeksekusi fungsi retry asinkron terlepas dari request context cancellation.
func GoDetachedWithRetry(ctx context.Context, cfg RetryConfig, fn func(bgCtx context.Context) error)

// NewSingleflight menginisialisasi Generic Deduplicator untuk mencegah Cache Stampede secara type-safe.
func NewSingleflight[T any]() *Singleflight[T]

// Do mengeksekusi fn hanya satu kali untuk key yang sama selama eksekusi masih berjalan,
// dan membagikan hasilnya ke seluruh pemanggil konkuren.
func (g *Singleflight[T]) Do(ctx context.Context, key string, fn func(ctx context.Context) (T, error)) (T, error, bool)

// DoChan mengeksekusi fn secara non-blocking dan mengembalikan channel penerima Result[T].
func (g *Singleflight[T]) DoChan(ctx context.Context, key string, fn func(ctx context.Context) (T, error)) <-chan Result[T]

// Forget menghapus key dari tabel aktif agar pemanggilan berikutnya memulai eksekusi baru.
func (g *Singleflight[T]) Forget(key string)
```

#### Contoh Penggunaan Generic Singleflight (Anti-Cache Stampede):
```go
type UserUsecase struct {
    repo   repository.UserRepository
    cache  cache.RedisClient
    sf     *routine.Singleflight[*entity.User]
}

func (u *UserUsecase) GetProfile(ctx context.Context, userID uint) (*entity.User, error) {
    cacheKey := fmt.Sprintf("user:profile:%d", userID)

    // 1. Cek cache terlebih dahulu
    if user, err := u.cache.Get(ctx, cacheKey); err == nil {
        return user, nil
    }

    // 2. Jika Cache MISS dan ada 1.000 request bersamaan, hanya 1 query yang ditembakkan ke DB!
    user, err, _ := u.sf.Do(ctx, cacheKey, func(execCtx context.Context) (*entity.User, error) {
        data, err := u.repo.FindByID(execCtx, userID)
        if err != nil {
            return nil, err
        }
        _ = u.cache.Set(execCtx, cacheKey, data, 10*time.Minute)
        return data, nil
    })

    return user, err
}
```

#### Contoh Penggunaan Worker Pool:
```go
// Inisialisasi pool dengan 4 worker, antrean 100, dan strategi blocking backpressure
pool := routine.NewPool(4, routine.WithQueueSize(100), routine.WithStrategy(routine.StrategyBlock))
defer pool.Stop()

// Submit tugas ke dalam antrean pool
err := pool.Submit(ctx, func(ctx context.Context) {
    processImageThumbnail(ctx, imageID)
})
```

#### Contoh Penggunaan Go with Retry:
```go
// Menjalankan background webhook dispatch dengan retry sampai 5 kali
routine.GoDetachedWithRetry(ctx, routine.RetryConfig{
    MaxAttempts:     5,
    InitialInterval: 200 * time.Millisecond,
    BackoffFactor:   2.0,
    Jitter:          true,
    RetryIf: func(err error) bool {
        // Hanya retry untuk network timeout atau 5xx HTTP
        return isTemporaryError(err)
    },
}, func(bgCtx context.Context) error {
    return webhookSender.Send(bgCtx, payload)
})
```

#### Prometheus Metrics yang Tersedia Otomatis di `/metrics`:
| Metric Name | Type | Labels | Keterangan |
|---|---|---|---|
| `routine_active_goroutines` | Gauge | - | Jumlah goroutine aktif yang sedang dikelola oleh `pkg/routine` |
| `routine_tasks_total` | Counter | `type` (`"go"`, `"safe"`, `"pool"`, `"retry"`, `"singleflight"`) | Akumulasi total tugas yang telah selesai dieksekusi |
| `routine_panics_total` | Counter | - | Akumulasi total panic yang berhasil ditangkap & dicegah dari crash |
| `routine_task_duration_seconds` | Histogram | `type` (`"go"`, `"safe"`, `"pool"`, `"retry"`, `"singleflight"`) | Distribusi durasi waktu eksekusi tugas |

## Contoh Package

### pagination (Offset & Cursor-Based Pagination Suite)

Menyediakan utilitas paginasi lengkap untuk ekosistem microservice:
1. **Offset-Based Pagination**: Cocok untuk Web Admin/CMS dengan navigasi nomor halaman dan status boolean `HasNext`/`HasPrev`.
2. **Cursor-Based Pagination**: Solusi O(1) untuk Infinite Scroll / Mobile Feeds jutaan baris tanpa masalah *offset drift* atau duplikasi data, menggunakan token opaque Base64 URL-safe.
3. **Dynamic Sorting & Whitelist Sanitizer**: Mengurai format pengurutan dinamis (`-created_at`, `name:desc`) dengan validasi whitelist ketat anti-SQL Injection.
4. **GORM Scope Helpers**: Mempercepat penulisan query di repository (`pagination.GormScope` dan `pagination.GormSortScope`).

```go
// pkg/pagination/pagination.go
package pagination

// Offset-Based
type Params struct {
    Page     int `json:"page"`
    PageSize int `json:"page_size"`
}

type Result[T any] struct {
    Items      []T   `json:"items"`
    TotalItems int64 `json:"total_items"`
    TotalPages int   `json:"total_pages"`
    Page       int   `json:"page"`
    PageSize   int   `json:"page_size"`
    HasNext    bool  `json:"has_next"`
    HasPrev    bool  `json:"has_prev"`
}

func NewResult[T any](items []T, totalItems int64, params Params) Result[T]

// Cursor-Based (pkg/pagination/cursor.go)
type CursorParams struct {
    Cursor string `json:"cursor"`
    Limit  int    `json:"limit"`
}

type CursorResult[T any] struct {
    Items      []T    `json:"items"`
    NextCursor string `json:"next_cursor,omitempty"`
    PrevCursor string `json:"prev_cursor,omitempty"`
    HasNext    bool   `json:"has_next"`
    HasPrev    bool   `json:"has_prev"`
    Limit      int    `json:"limit"`
}

func EncodeCursor(id any, createdAt ...time.Time) string
func DecodeCursor(token string) (*CursorData, error)

// Sorting (pkg/pagination/sort.go)
func ParseSort(query string, defaultField string, defaultOrder OrderDirection) Sort
func (s Sort) SQL(whitelist map[string]string) string

// GORM Scopes (pkg/pagination/gorm.go)
func GormScope(p Params) func(db *gorm.DB) *gorm.DB
func GormSortScope(s Sort, whitelist map[string]string) func(db *gorm.DB) *gorm.DB
```

#### Contoh Penggunaan di Repository (Offset + GORM Scopes):
```go
func (r *UserRepository) ListUsers(ctx context.Context, p pagination.Params, s pagination.Sort) (pagination.Result[entity.User], error) {
    allowedCols := map[string]string{
        "id":         "users.id",
        "name":       "users.name",
        "created_at": "users.created_at",
    }

    var users []entity.User
    var total int64

    db := postgres.GetDB(ctx, r.db)
    if err := db.Model(&entity.User{}).Count(&total).Error; err != nil {
        return pagination.Result[entity.User]{}, err
    }

    err := db.Scopes(
        pagination.GormScope(p),
        pagination.GormSortScope(s, allowedCols),
    ).Find(&users).Error

    return pagination.NewResult(users, total, p), err
}
```

#### Contoh Penggunaan Cursor-Based Pagination (Infinite Scroll):
```go
func (r *OrderRepository) ListOrdersCursor(ctx context.Context, p pagination.CursorParams) (pagination.CursorResult[entity.Order], error) {
    p.Normalize()
    cursor, err := pagination.DecodeCursor(p.Cursor)
    if err != nil {
        return pagination.CursorResult[entity.Order]{}, err
    }

    query := postgres.GetDB(ctx, r.db).Model(&entity.Order{})
    if cursor != nil && cursor.CreatedAt != nil {
        query = query.Where("created_at < ? OR (created_at = ? AND id < ?)", cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
    }
    query = query.Order("created_at DESC, id DESC").Limit(p.Limit + 1) // Probe +1 to detect hasNext

    var orders []entity.Order
    if err := query.Find(&orders).Error; err != nil {
        return pagination.CursorResult[entity.Order]{}, err
    }

    hasMore := len(orders) > p.Limit
    return pagination.NewCursorResult(orders, p.Limit, hasMore, func(o entity.Order) string {
        return pagination.EncodeCursor(o.ID, o.CreatedAt)
    }), nil
}
```

### httpclient (Resilient Outbound HTTP Client)

Menyediakan HTTP client siap produksi untuk pemanggilan API eksternal / third-party / microservice lain:
1. **Tracing W3C Otomatis**: Otomatis menyuntikkan header `traceparent` dari context OpenTelemetry.
2. **Korelasi `X-Request-ID`**: Menyuntikkan `X-Request-ID` secara otomatis dari `contextutil`.
3. **Safe Connection Pooling**: `MaxIdleConns: 100`, `MaxIdleConnsPerHost: 20`, `IdleConnTimeout: 90s` (mencegah socket exhaustion).
4. **Retry Transient Failure**: Dukungan retry dengan *exponential backoff* dan *jitter* untuk error 502/503/504 atau network timeout.
5. **HTTP QUERY Method (IETF RFC 10008)**: Mendukung method resmi `QUERY` yang aman (*safe*) dan idempoten untuk query berbadan kompleks (*request body*) tanpa batasan panjang URI.
6. **JSON Convenience Methods**: `GetJSON`, `PostJSON`, dan `QueryJSON` untuk mempermudah konsumsi REST API.

```go
// pkg/httpclient/client.go
package httpclient

client := httpclient.New(
    httpclient.WithTimeout(10 * time.Second),
    httpclient.WithRetry(3, 100*time.Millisecond, 2*time.Second),
    httpclient.WithLogger(logger),
)

// Mengonsumsi API eksternal dengan otomatisasi tracing dan JSON parsing:
var target PaymentStatusResponse
resp, err := client.PostJSON(ctx, "https://api.payment.com/v1/charges", chargeReq, &target)

// Menggunakan HTTP QUERY (RFC 10008) untuk pencarian/filter kompleks dengan request body:
var searchResult SearchResult
filter := SearchFilter{Tags: []string{"golang", "microservice"}, MinPrice: 10000}
resp, err = client.QueryJSON(ctx, "https://api.search.com/v1/items", filter, &searchResult)
```

### pointer

```go
// pkg/pointer/pointer.go
package pointer

func Of[T any](v T) *T {
    return &v
}

func ValueOrDefault[T any](p *T, def T) T {
    if p == nil {
        return def
    }
    return *p
}
```

### errorutil

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

func New(code Code, message string) *AppError {
    return &AppError{Code: code, Message: message}
}

func Wrap(code Code, message string, err error) *AppError {
    return &AppError{Code: code, Message: message, Err: err}
}
```

## Dependency Direction

```
internal ──────► pkg
infrastructure ► pkg
bootstrap ─────► pkg

pkg ──────────── TIDAK BOLEH import internal, infrastructure, atau bootstrap
```

## Rules

1. **Jangan taruh business logic di `pkg/`.** Jika sebuah helper hanya relevan untuk satu feature, taruh di dalam feature tersebut.
2. **Jangan taruh framework-specific code di `pkg/`.** Wrapper untuk GORM, Redis, dsb tetap di `infrastructure/`.
3. **Setiap sub-package harus independent.** `pkg/pagination` tidak boleh import `pkg/errorutil`.
4. **Naming harus generic.** Hindari nama yang mengandung konteks feature (misalnya `pkg/userutil` — salah).
5. **Minimal dependency.** Package di `pkg/` idealnya hanya bergantung pada standard library Go.

---

# Proto Layout

```text
api/proto/
├── user/
│   └── v1/
│       └── user.proto
└── auth/
    └── v1/
        └── auth.proto
```

Generated:

```text
gen/proto/
├── user/
│   └── v1/
│       ├── user.pb.go
│       ├── user_grpc.pb.go
│       └── user.pb.gw.go
└── auth/
    └── v1/
        ├── auth.pb.go
        ├── auth_grpc.pb.go
        └── auth.pb.gw.go
```

Gunakan versioning sejak awal:

```
user/v1
user/v2
```

---

# Testing

## Unit Test

Disimpan berdampingan dengan source:

```text
usecase/
├── create.go
└── create_test.go
```

---

## Integration Test

```text
tests/integration/
```

Menggunakan Docker Database (testcontainers-go recommended).

---

## End-to-End Test

```text
tests/e2e/
```

Menjalankan service secara utuh.

---

## Mock

Mock di-generate menggunakan tool seperti `mockgen` atau `mockery`.

Generated mock **tidak** disimpan di repository. Tambahkan ke `.gitignore`:

```gitignore
# Generated mocks
**/mock_*.go
**/mocks/
```

Generate mock via Makefile target:

```makefile
.PHONY: mocks
mocks:
	go generate ./internal/...
```

Gunakan `//go:generate` directive di interface file:

```go
// internal/user/repository/interface.go
//go:generate mockgen -source=interface.go -destination=mock_repository.go -package=repository

type Repository interface {
    FindByID(ctx context.Context, id uint) (*entity.User, error)
    Create(ctx context.Context, user *entity.User) error
}
```

---

# Naming Convention

Gunakan package kecil:

```
entity
repository
handler
usecase
```

Struct di dalam package menggunakan nama **deskriptif berdasarkan use case**, bukan prefixed dengan nama feature:

| Salah (redundant)    | Benar                 |
| -------------------- | --------------------- |
| `UserRepository`     | `Repository`          |
| `UserHandler`        | `Handler`             |
| `UserUsecase`        | `CreateUsecase`       |

Untuk usecase, gunakan nama berdasarkan aksi:

```go
// internal/user/usecase/
CreateUsecase
FindUsecase
UpdateUsecase
DeleteUsecase
```

Struct naming selalu `Usecase`, bukan `Service`:

```go
type CreateUsecase struct {
    repo repository.Repository
}
```

---

# Google Wire

Setiap Feature memiliki Provider sendiri.

## Basic Provider

```go
// internal/user/provider.go
package user

import (
    "github.com/google/wire"
    "my-service/internal/user/handler"
    "my-service/internal/user/usecase"
    "my-service/internal/user/repository"
)

var ProviderSet = wire.NewSet(
    usecase.NewCreateUsecase,
    usecase.NewFindUsecase,
    handler.NewHandler,
    // Bind interface ke implementation
    wire.Bind(new(repository.Repository), new(*postgres.UserRepository)),
)
```

## Infrastructure Provider

```go
// infrastructure/database/postgres/provider.go
package postgres

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
    NewUserRepository,
    NewOrderRepository,
)
```

## Bootstrap Wire

```go
// bootstrap/wire.go
//go:build wireinject

package bootstrap

import (
    "github.com/google/wire"
    "my-service/internal/user"
    "my-service/internal/auth"
    "my-service/infrastructure/database/postgres"
    infraConfig "my-service/infrastructure/config"
)

func InitializeApp(configPath string) (*App, error) {
    wire.Build(
        infraConfig.ProviderSet,
        postgres.ProviderSet,
        user.ProviderSet,
        auth.ProviderSet,
        NewApp,
    )
    return nil, nil
}
```

---

# Observability

## Prinsip

Observability diimplementasikan di infrastructure layer dan di-inject sebagai cross-cutting concern melalui middleware/interceptor. Business logic **tidak** memanggil logger atau tracer secara langsung kecuali untuk domain-specific logging.

## Structured Logging

Gunakan standardized microservice structured JSON logger yang membungkus Uber Zap (`pkg/logger`):

* **Format 100% JSON**: Output selalu berupa JSON terstruktur (ISO8601 UTC timestamp, level uppercase, short caller `file:line`).
* **Zero Parameter Variadic Field**: Method log (`Info`, `Warn`, `Error`, `Debug`) tidak menerima `fields ...Field`. Seluruh field korelasi microservice diekstrak secara otomatis dari `context.Context`.
* **Deep gRPC Middleware Integration**: `LoggingInterceptor` menyuntikkan `RPCMetadata` (`rpc.system`, `rpc.service`, `rpc.method`, `client_ip`, `user_agent`) dan `request_id` ke dalam context sehingga log di layer usecase otomatis mewarisi konteks RPC.
* **Audit Payload Policy**:
  - `request`: Selalu dicatat (dengan masking data sensitif seperti password, token, card number).
  - `response`: **Hanya dicatat ketika terjadi error (`code != codes.OK`)** untuk menjaga efisiensi I/O storage dan CPU di production.

### Standar Field Log JSON

| JSON Key | Tipe | Sumber | Deskripsi |
| :--- | :--- | :--- | :--- |
| `timestamp` | string (ISO8601 UTC) | Zap Core | e.g. `2026-09-10T16:30:00.123Z` |
| `level` | string | Zap Core | `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `msg` | string | Parameter | Pesan log |
| `caller` | string | Caller Encoder | `usecase/user.go:45` (CallerSkip 1) |
| `service` | string | App Config | Nama service |
| `env` | string | App Config | `development`, `staging`, `production` |
| `host` | string | OS Hostname | Container/Pod hostname |
| `trace_id` | string | OpenTelemetry SpanContext | W3C Distributed Trace ID |
| `span_id` | string | OpenTelemetry SpanContext | Current Span ID |
| `request_id` | string | Context / gRPC Metadata | Correlation request ID |
| `user_id` | uint | Context (`contextutil`) | ID authenticated user |
| `role` | string | Context (`contextutil`) | Role authenticated user |
| `client_ip` | string | Context / Peer | IP caller |
| `user_agent` | string | Context / Metadata | Client caller agent |
| `rpc.system` | string | gRPC Interceptor | `"grpc"` |
| `rpc.service` | string | gRPC Interceptor | Service gRPC target |
| `rpc.method` | string | gRPC Interceptor | Method gRPC target |
| `rpc.grpc.status_code`| int | gRPC Interceptor | Kode status gRPC numerik |
| `status` | string | gRPC Interceptor | Nama status (e.g. `OK`, `Internal`) |
| `duration_ms` | float64 | gRPC Interceptor | Durasi eksekusi dalam ms |
| `request` | object | gRPC Interceptor | Sanitized request payload |
| `response` | object | gRPC Interceptor | Sanitized response (hanya saat error) |
| `error` | string | Parameter Error | Pesan error |
| `data` | any | Fluent `.WithData()` | Custom structured payload |

### Penggunaan di Business Logic (Usecase / Handler)

```go
// Bersih tanpa perlu passing fields manual; context otomatis mengekstrak trace_id, user_id, rpc.*
logger.Info(ctx, "user profile updated successfully")

// Jika ada error:
if err != nil {
    logger.Error(ctx, "failed to persist transaction to database", err)
    return err
}

// Menambahkan data payload terstruktur opsional:
logger.WithData(paymentResponse).Info(ctx, "received payment gateway confirmation")
```

## Distributed Tracing

Gunakan OpenTelemetry:

```go
// infrastructure/telemetry/tracer.go
package telemetry

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func NewTracer(config Config) (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(config.OTLPEndpoint),
    )
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName(config.ServiceName),
        )),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

Trace ID propagation via gRPC interceptor:

```go
// bootstrap/grpc.go
import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

server := grpc.NewServer(
    grpc.StatsHandler(otelgrpc.NewServerHandler()),
)
```

### Full-Stack APM Trace Waterfall & Instrumentasi Komponen

Arsitektur boilerplate ini mengintegrasikan tracing terdistribusi secara *end-to-end* (APM):

```text
[Client / Browser]
        │
        ▼ (W3C traceparent header)
[HTTP Gateway :8080] ────── otelhttp.NewHandler (Ingress Span)
        │
        ▼ (gRPC Metadata)
[gRPC Server :50051] ────── otelgrpc.NewServerHandler (Server Span)
        │
        ├──► [GORM PostgreSQL] ── gorm.io/plugin/opentelemetry (DB Child Span + Query)
        ├──► [Redis Cache]     ── redisotel.InstrumentTracing (Cache Child Span + Command)
        └──► [External HTTP]   ── pkg/httpclient (W3C Traceparent Injected Outbound Span)
```

1. **HTTP Ingress Tracing (`otelhttp`)**: Gateway REST (`bootstrap/gateway.go`) membungkus handler HTTP dengan `otelhttp.NewHandler`, otomatis mengekstrak W3C `traceparent` dari klien atau memulai trace root baru.
2. **gRPC Transport Tracing (`otelgrpc`)**: Server gRPC (`bootstrap/grpc.go`) mencatat span RPC dan meneruskan konteks trace ke seluruh handler dan usecase.
3. **Database Query Tracing (`gorm.io/plugin/opentelemetry`)**: GORM (`infrastructure/database/postgres/db.go`) mencatat eksekusi query SQL sebagai child span lengkap dengan durasi dan query string.
4. **Cache Command Tracing (`redisotel`)**: Redis (`infrastructure/cache/redis/redis.go`) melacak operasi Redis (GET, SET, DEL, dll.) sebagai child span.
5. **Outbound HTTP Tracing (`pkg/httpclient`)**: Otomatis menginjeksikan header `traceparent` saat memanggil layanan eksternal.
6. **Continuous Profiling (`net/http/pprof`)**: Disajikan di `/debug/pprof/` pada environment non-production untuk profiling CPU, memory allocations, goroutine leaks, dan mutex contention.


## Metrics

Gunakan Prometheus via OpenTelemetry:

```go
// infrastructure/telemetry/metrics.go
package telemetry

import (
    "go.opentelemetry.io/otel/exporters/prometheus"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func NewMeter(config Config) (*sdkmetric.MeterProvider, error) {
    exporter, err := prometheus.New()
    if err != nil {
        return nil, err
    }
    mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
    return mp, nil
}
```

Metric naming convention:

```
<service>_<feature>_<action>_<unit>

Contoh:
myservice_user_created_total
myservice_order_processing_duration_seconds
myservice_auth_login_failed_total
```

## Middleware Integration

Semua cross-cutting concern di-wire via gRPC unary interceptor chain dengan urutan terencana:

```go
server := grpc.NewServer(
    grpc.StatsHandler(otelgrpc.NewServerHandler()), // distributed tracing
    grpc.ChainUnaryInterceptor(
        middleware.RecoveryInterceptor(logger),                  // 1. panic recovery terluar
        middleware.LoggingInterceptor(logger),                   // 2. access log & context enrichment
        middleware.TimeoutInterceptor(cfg.Server.DefaultTimeout), // 3. request deadline enforcement
        middleware.AuthInterceptor(cfg),                         // 4. jwt authentication
        middleware.ErrorInterceptor(),                           // 5. domain error to grpc status mapping
    ),
)
```

### Deadline & Request Timeout (`TimeoutInterceptor`)
- **Default Server Timeout**: Diatur melalui `SERVER_DEFAULT_TIMEOUT` (default: `15s`).
- **Aturan Preseden**:
  1. Jika klien memberikan deadline yang lebih ketat, deadline klien dipertahankan.
  2. Jika klien tidak memberikan deadline atau memberikan deadline lebih longgar dari server, batas server dipaksakan via `context.WithTimeout`.
  3. Mendukung per-method custom overrides untuk RPC query berat atau ekspor data.
- **Mapping Error**: Timeout menghasilkan gRPC `codes.DeadlineExceeded` (dipetakan gateway ke HTTP `504 Gateway Timeout`). Pembatalan oleh klien menghasilkan `codes.Canceled` (HTTP `499 Client Closed Request`).

---

# Health Check & Graceful Shutdown

## Health Check

Implement gRPC Health Checking Protocol:

```go
// bootstrap/grpc.go
import "google.golang.org/grpc/health"
import healthpb "google.golang.org/grpc/health/grpc_health_v1"

healthServer := health.NewServer()
healthpb.RegisterHealthServer(grpcServer, healthServer)

// Set service status
healthServer.SetServingStatus("user.v1.UserService", healthpb.HealthCheckResponse_SERVING)
```

Untuk HTTP health check (Kubernetes readiness/liveness):

```go
// bootstrap/gateway.go
mux.HandlePath("GET", "/healthz", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
})

mux.HandlePath("GET", "/readyz", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
    // Check database connectivity, etc.
    if err := db.PingContext(r.Context()); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
})
```

## Graceful Shutdown

```go
// cmd/api/main.go
func main() {
    app, cleanup, err := bootstrap.InitializeApp("configs/config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    defer cleanup()

    // Start servers
    go app.StartGRPC()
    go app.StartHTTPGateway()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("shutting down...")

    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    app.Shutdown(ctx)
}
```

```go
// bootstrap/app.go
func (a *App) Shutdown(ctx context.Context) {
    // 1. Stop accepting new requests
    a.grpcServer.GracefulStop()

    // 2. Shutdown HTTP gateway
    a.httpServer.Shutdown(ctx)

    // 3. Close database connections
    a.db.Close()

    // 4. Flush telemetry
    a.tracerProvider.Shutdown(ctx)
    a.meterProvider.Shutdown(ctx)

    // 5. Flush logger
    a.logger.Sync()
}
```

---

# Adding a New Feature (Checklist)

Checklist ini memastikan setiap tim menambahkan feature dengan cara yang konsisten dan minim conflict.

## Step-by-Step

1. **Buat folder feature di `internal/`**

```text
internal/<feature_name>/
├── dto/
├── entity/
├── errors/
├── handler/
├── repository/
│   └── interface.go
├── usecase/
├── validator/
└── provider.go
```

2. **Definisikan entity dan repository interface**

```go
// internal/<feature>/entity/<feature>.go
// internal/<feature>/repository/interface.go
```

3. **Implementasikan usecase**

```go
// internal/<feature>/usecase/create.go
// internal/<feature>/usecase/create_test.go
```

4. **Buat proto definition**

```text
api/proto/<feature>/v1/<feature>.proto
```

Sertakan grpc-gateway annotations untuk REST endpoint.

5. **Generate proto files**

```bash
make proto
```

6. **Implementasikan handler (gRPC server)**

```go
// internal/<feature>/handler/handler.go
```

7. **Implementasikan repository di infrastructure**

```go
// infrastructure/database/postgres/<feature>.go
```

8. **Buat Wire provider**

```go
// internal/<feature>/provider.go
var ProviderSet = wire.NewSet(...)
```

9. **Register di bootstrap**

```go
// bootstrap/wire.go → tambahkan ProviderSet
// bootstrap/gateway.go → register gateway handler
```

10. **Buat migration (jika ada schema baru)**

```bash
make migration name=create_<feature>_table
```

11. **Tambahkan `//go:generate` untuk mock**

12. **Jalankan test**

```bash
make test
make lint
```

## File yang Akan Conflict (Shared Files)

| File | Perubahan | Conflict Risk |
|------|-----------|---------------|
| `bootstrap/wire.go` | +1 baris (import + ProviderSet) | Low |
| `bootstrap/gateway.go` | +1 baris (RegisterHandler) | Low |
| `infrastructure/database/postgres/provider.go` | +1 baris (NewRepository) | Low |
| `Makefile` | Biasanya tidak berubah | Very Low |

Tips: Lakukan perubahan di shared files sebagai commit terpisah agar mudah di-rebase jika conflict.

---

# Migration Convention

## Naming

Gunakan format timestamp-based untuk menghindari conflict antar tim:

```
<YYYYMMDD>_<sequence>_<description>.sql
```

Contoh:

```text
migrations/
├── 20260101_001_create_users_table.sql
├── 20260101_002_create_auth_tokens_table.sql
├── 20260215_001_create_orders_table.sql
├── 20260215_002_add_status_to_orders.sql
└── 20260301_001_create_payments_table.sql
```

Timestamp-based naming memastikan dua tim yang bekerja paralel tidak pernah conflict pada nomor urut.

## Ownership

Setiap migration harus di-own oleh tim yang memiliki feature terkait:

```sql
-- Migration: 20260215_001_create_orders_table.sql
-- Owner: Team Order
-- Feature: order

CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    total_amount DECIMAL(12,2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## Rules

1. **Satu migration, satu concern.** Jangan campur perubahan schema dari feature berbeda dalam satu file.
2. **Backward compatible.** Migration tidak boleh break existing data atau existing queries yang sedang berjalan.
3. **Idempotent.** Gunakan `IF NOT EXISTS`, `IF EXISTS` untuk safety.
4. **No destructive changes tanpa review.** `DROP TABLE`, `DROP COLUMN`, rename column harus melalui multi-step migration:
   - Step 1: Tambah kolom/tabel baru
   - Step 2: Migrate data
   - Step 3: Update code untuk pakai yang baru
   - Step 4: Hapus yang lama (di release berikutnya)

5. **Jangan edit migration yang sudah di-merge ke main.** Buat migration baru untuk fix.

## Makefile Target

```makefile
.PHONY: migration
migration:
	@read -p "Migration name: " name; \
	touch migrations/$$(date +%Y%m%d)_001_$$name.sql

.PHONY: migrate-up
migrate-up:
	migrate -path migrations -database $(DATABASE_URL) up

.PHONY: migrate-down
migrate-down:
	migrate -path migrations -database $(DATABASE_URL) down 1
```

---

# Feature Removal Checklist

Ketika feature sudah tidak digunakan atau dipindahkan ke service lain, ikuti checklist ini untuk memastikan removal yang bersih.

## Step-by-Step

1. **Deprecation notice (minimal 1 sprint sebelum removal)**

   - Tandai proto endpoint dengan `deprecated = true`:
   ```protobuf
   rpc GetLegacyUser(GetUserRequest) returns (GetUserResponse) {
     option deprecated = true;
   }
   ```
   - Tambahkan log warning saat endpoint dipanggil
   - Komunikasikan ke tim lain yang mungkin bergantung

2. **Verifikasi tidak ada consumer**

   - Cek metrics: apakah endpoint masih menerima traffic?
   - Cek cross-feature dependency: apakah ada feature lain yang inject interface dari feature ini?
   - Cek service lain: apakah ada gRPC client yang masih memanggil?

3. **Hapus code (dalam urutan ini)**

   | Urutan | Yang Dihapus | File |
   |--------|-------------|------|
   | 1 | Unregister dari gateway | `bootstrap/gateway.go` |
   | 2 | Hapus dari Wire | `bootstrap/wire.go` |
   | 3 | Hapus infrastructure impl | `infrastructure/database/postgres/<feature>.go` |
   | 4 | Hapus feature folder | `internal/<feature>/` |
   | 5 | Hapus proto | `api/proto/<feature>/` |
   | 6 | Hapus generated code | `gen/proto/<feature>/` |
   | 7 | Buat migration (drop table) | `migrations/` |

4. **Buat migration untuk cleanup schema**

   ```sql
   -- Migration: 20260701_001_drop_legacy_feature_table.sql
   -- Owner: Team Platform
   -- Reason: Feature moved to separate service

   DROP TABLE IF EXISTS legacy_feature;
   ```

5. **Update documentation**

   - Hapus dari README atau API docs
   - Update changelog

6. **Run full test suite**

   ```bash
   make test
   make integration-test
   ```

## Yang Sering Terlewat

- [ ] Hapus config entries di `configs/` yang khusus untuk feature tersebut
- [ ] Hapus environment variables terkait
- [ ] Hapus Kubernetes secrets/configmaps yang khusus feature
- [ ] Update monitoring dashboard (hapus panel yang sudah tidak relevan)
- [ ] Hapus CI/CD steps yang khusus feature (jika ada)

---

# Design Philosophy

* Feature First
* Clean Architecture
* Business Independent
* Framework Independent
* Database Independent
* Transport Independent
* Microservice Ready
* Testable
* Replaceable Infrastructure
* Single Responsibility
* Generated Code Separated
* Public API Contract Separated
* Explicit Dependency Direction

---

# Dependency Direction

```
cmd
 │
 ▼
bootstrap
 │
 ▼
internal (business logic + interfaces)
 │
 ▼
infrastructure (implements interfaces)
 │
 ├──────────────┐
 ▼              ▼
Database    External Service

        ┌──────────────────┐
        │       pkg        │
        │ (shared utility) │
        └──────────────────┘
              ▲  ▲  ▲
              │  │  │
   internal ──┘  │  └── infrastructure
                 │
            bootstrap
```

Tidak boleh ada dependency yang mengarah kembali ke atas.

---

# Summary

| Folder           | Purpose                              |
| ---------------- | ------------------------------------ |
| `cmd`            | Application Entry Point              |
| `bootstrap`      | Composition Root + Server Setup      |
| `internal`       | Business Logic + Domain Interfaces   |
| `infrastructure` | Framework & Adapter Implementation   |
| `pkg`            | Shared Library, Helper, Wrapper      |
| `api`            | API Contract (Proto + Gateway Annot) |
| `gen`            | Generated Code (proto, gateway, mock)|
| `configs`        | Configuration Files                  |
| `migrations`     | Database Migration                   |
| `tests`          | Integration & E2E Test               |

Blueprint ini ditujukan untuk membangun microservice Go yang konsisten, mudah diuji, mudah dikembangkan, dan siap berkembang dari layanan sederhana hingga sistem enterprise tanpa perlu mengubah struktur dasar proyek.