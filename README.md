# Go Feature-Based Modular Clean Architecture Boilerplate

Standard production-ready microservice boilerplate untuk Go yang menerapkan **Feature-Based Modular Clean Architecture** dengan prinsip Hexagonal Architecture (Ports & Adapters) dan pendekatan Lightweight Domain-Driven Design (DDD).

Boilerplate ini menyelaraskan pemisahan logika bisnis dari infrastruktur agar aplikasi tetap modular, mudah diuji (*testable*), dan independen terhadap teknologi eksternal.

---

## 📋 Daftar Isi

- [Fitur Utama](#-fitur-utama)
- [Arsitektur & Prinsip Desain](#-arsitektur--prinsip-desain)
- [Struktur Proyek](#-struktur-proyek)
- [Prasyarat Sistem & Tooling](#-prasyarat-sistem--tooling)
- [Daftar Dependensi (Libraries & Frameworks)](#-daftar-dependensi-libraries--frameworks)
- [Panduan Memulai Cepat (Quick Start)](#-panduan-memulai-cepat-quick-start)
- [Konfigurasi Environment (.env)](#-konfigurasi-environment-env)
- [Manajemen Command (Makefile)](#-manajemen-command-makefile)
- [Spesifikasi API & Endpoints](#-spesifikasi-api--endpoints)
- [Generasi Kode (Code Generation)](#-generasi-kode-code-generation)
- [Pengujian & Kualitas Kode](#-pengujian--kualitas-kode)
- [Dokumentasi Tambahan](#-dokumentasi-tambahan)

---

## ✨ Fitur Utama

- **Feature-First Architecture**: Kode bisnis dikelompokkan berdasarkan *domain feature* (contoh: `internal/auth`, `internal/user`, `internal/health`), bukan layer global monolithic.
- **Dual Protocol Support**:
  - **gRPC Server** berjalan pada port `:50051`.
  - **HTTP REST Gateway** (menggunakan `grpc-gateway`) berjalan pada port `:8080`.
- **Dependency Injection (Google Wire)**: Injeksi dependensi bertipe compile-time aman menggunakan Google Wire.
- **Swagger / OpenAPI 2.0**: Swagger UI disajikan langsung secara native di endpoint `/swagger/`.
- **Autentikasi & Keamanan**:
  - JWT Access Token & Refresh Token lifecycle.
  - Password hashing dengan `bcrypt`.
- **Database & Caching**:
  - **PostgreSQL**: Primary relational database via GORM dengan connection pooling and automated schema migrations (`golang-migrate`).
  - **Modular Extensions (Redis & MongoDB)**: Client driver untuk Redis dan MongoDB telah tersedia di `infrastructure/cache/redis` dan `infrastructure/database/mongo`, siap dihubungkan ke `bootstrap/wire.go` bila fitur baru memerlukan caching profil atau document storage.
- **Observability & Security**:
  - Structured Logging (**Uber Zap**) terkorelasi dengan OpenTelemetry trace & span ID.
  - Metrics Prometheus disajikan di `/metrics`.
  - Token refresh hashing (SHA-256) & Token Reuse Compromise Detection (RFC 6819).
  - CORS middleware aktif pada HTTP gateway.
  - Graceful Shutdown untuk HTTP gateway dan gRPC server.
- **Container Ready**: Dilengkapi Dockerfile multi-stage build dan `docker-compose.yml` untuk lingkungan pengembangan lokal.

---

## 🏛 Arsitektur & Prinsip Desain

Aplikasi ini menggunakan perpaduan dari Clean Architecture & Hexagonal Architecture:

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

1. **Separation of Concerns**: Logika bisnis tidak bergantung pada framework HTTP, database GORM, Redis, atau gRPC. Semuanya terhubung via interface.
2. **Feature First**: Setiap fitur di bawah `internal/` mengemas `dto`, `entity`, `errors`, `handler`, `repository`, `usecase`, dan `validator` masing-masing.

---

## 📂 Struktur Proyek

```text
.
├── api/
│   └── proto/             # Proto definitions (.proto) per fitur (auth, user, health)
├── bootstrap/             # Dependency Injection Wire & Server Bootstrapping
│   ├── app.go             # Lifecycle & Server Runner (gRPC + HTTP)
│   ├── gateway.go         # Setup HTTP REST Gateway & Swagger UI
│   ├── grpc.go            # Setup gRPC Server & Interceptors
│   ├── wire.go            # Wire Injector Declaration
│   └── wire_gen.go        # Generated Wire Code
├── cmd/
│   └── api/
│       └── main.go        # Entrypoint utama aplikasi
├── deployments/           # Dockerfile & Docker Compose configuration
│   ├── Dockerfile
│   └── docker-compose.yml
├── docs/                  # Dokumentasi arsitektur detail (architecture.md)
├── gen/                   # Code yang digenerasi otomatis dari Proto & OpenAPI
│   ├── openapi/           # Embedded OpenAPI JSON Spec
│   └── pb/                # Generated Go Protocol Buffer code
├── infrastructure/        # Implementasi Driver & Framework
│   ├── cache/             # Redis Client
│   ├── config/            # Viper Config Loader
│   ├── database/          # GORM PostgreSQL & MongoDB connection
│   ├── logger/            # Zap Logger provider
│   ├── middleware/        # gRPC Interceptors (Logging, Recovery, Auth)
│   ├── swagger/           # Embedded Swagger UI handler
│   └── telemetry/         # OpenTelemetry Tracer & Meter Providers
├── internal/              # Logika Bisnis (Domain Features)
│   ├── auth/              # Fitur Autentikasi (Login, Refresh, Logout)
│   ├── health/            # Probes Health Check (Liveness & Readiness)
│   └── user/              # Fitur Management User (CRUD)
├── migrations/            # Script Migrasi Database SQL
├── pkg/                   # Helper & Utility Generic (crypto, pagination, validator, dll)
└── scripts/               # Script utilitas (protogen.sh / protogen.bat)
```

---

## 🛠 Prasyarat Sistem & Tooling

Sebelum menjalankan aplikasi, pastikan sistem Anda telah terpasang perangkat lunak berikut:

### 1. Kebutuhan Sistem Utama
- **Go**: `v1.25` atau lebih baru
- **Docker** & **Docker Compose**
- **GNU Make** (utility untuk menjalankan target `Makefile`)
- **protoc** (Protocol Buffers Compiler v3+) — *opsional, hanya jika ingin mengompilasi ulang file `.proto`*

### 2. CLI Development Tools
Untuk menjalankan seluruh target otomatisasi (`make migrate-*`, `make wire`, `make lint`, `make mocks`), pasang CLI tools berikut:

```bash
# 1. Database Migration Tool (golang-migrate dengan driver postgres)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 2. Dependency Injection Generator (Google Wire)
go install github.com/google/wire/cmd/wire@latest

# 3. Linter Standar Go
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 4. Mock Generator untuk Unit Test
go install go.uber.org/mock/mockgen@latest

# 5. Protoc Plugins (Hanya jika mengubah skema .proto)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

---

## 📦 Daftar Dependensi (Libraries & Frameworks)

Boilerplate ini menggunakan dependensi pihak ketiga pilihan yang telah teruji kestabilan dan performanya di lingkungan produksi:

| Kategori | Package / Library | Versi | Peran & Kegunaan |
| :--- | :--- | :--- | :--- |
| **Transport & API** | `google.golang.org/grpc` | `v1.83.0` | Server & client framework RPC berperforma tinggi |
| | `github.com/grpc-ecosystem/grpc-gateway/v2` | `v2.29.0` | Reverse-proxy otomatis gRPC ke HTTP RESTful JSON API |
| | `google.golang.org/protobuf` | `v1.36.11` | Runtime encoder/decoder Protocol Buffers |
| **Dependency Injection** | `github.com/google/wire` | `v0.7.0` | Compile-time dependency injection aman tanpa reflection runtime |
| **Database & Persistence**| `gorm.io/gorm` | `v1.31.2` | Developer-friendly ORM untuk pemetaan database |
| | `gorm.io/driver/postgres` | `v1.6.0` | Driver PostgreSQL berbasis `pgx/v5` connection pool |
| | `go.mongodb.org/mongo-driver` | `v1.17.9` | Driver resmi MongoDB untuk penyimpanan dokumen NoSQL |
| **Caching** | `github.com/redis/go-redis/v9` | `v9.21.0` | Client Redis untuk caching performa tinggi & rate-limiting |
| **Konfigurasi** | `github.com/spf13/viper` | `v1.21.0` | Pembaca konfigurasi environment variables & file `.env` |
| **Observability** | `go.uber.org/zap` | `v1.28.0` | Structured logger berkecepatan tinggi dengan alokasi memori minimal |
| | `go.opentelemetry.io/otel` | `v1.45.0` | Standar telemetri terdistribusi (Distributed Tracing OTLP) |
| | `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` | `v0.69.0` | Interceptor tracing otomatis untuk gRPC server |
| | `github.com/prometheus/client_golang` | `v1.24.1` | Metrik Prometheus yang diekspos melalui endpoint `/metrics` |
| **Keamanan & Auth** | `github.com/golang-jwt/jwt/v5` | `v5.3.1` | Penandatanganan & verifikasi JWT Access & Refresh Token (RFC 7519) |
| | `golang.org/x/crypto/bcrypt` | `v0.54.0` | Algoritma hashing password searah dengan adaptive cost |
| **Validasi Data** | `github.com/go-playground/validator/v10` | `v10.30.3` | Validasi struct request payload berbasis struct tag |
| **Testing** | `github.com/stretchr/testify` | `v1.11.1` | Toolkit assertion dan mocking untuk pengujian unit |

---

## 🚀 Panduan Memulai Cepat (Quick Start)

### 1. Clone & Setup Environment

Salin file konfigurasi environment sampel ke `.env`:

```bash
cp .env.example .env
```

### 2. Jalankan Infrastruktur Lokal (Database, Redis, Jaeger)

Gunakan Docker Compose untuk menjalankan PostgreSQL, MongoDB, Redis, OTel Collector, dan Jaeger:

```bash
make docker-up
```

### 3. Jalankan Migrasi Database

Aplikasikan migrasi tabel awal ke PostgreSQL:

```bash
make migrate-up
```

### 4. Jalankan Aplikasi

Jalankan aplikasi dalam mode lokal:

```bash
make run
```

Aplikasi akan berjalan dan mendengarkan port berikut:
- **gRPC Server**: `localhost:50051`
- **HTTP Gateway**: `http://localhost:8080`
- **Swagger UI**: `http://localhost:8080/swagger/`

---

## ⚙️ Konfigurasi Environment (.env)

Konfigurasi utama dimuat dari variabel lingkungan atau file `.env`:

| Variabel | Default | Deskripsi |
| --- | --- | --- |
| `APP_NAME` | `go-feature-based-boilerplate` | Nama aplikasi |
| `APP_ENV` | `development` | Environment (`development`, `production`, `test`) |
| `SERVER_GRPC_PORT` | `50051` | Port untuk gRPC Server |
| `SERVER_HTTP_PORT` | `8080` | Port untuk HTTP REST Gateway |
| `DB_HOST` | `localhost` | Host PostgreSQL |
| `DB_PORT` | `5432` | Port PostgreSQL |
| `DB_NAME` | `appdb` | Nama database PostgreSQL |
| `DB_USER` | `appuser` | Username PostgreSQL |
| `DB_PASSWORD` | `apppassword` | Password PostgreSQL |
| `REDIS_ADDR` | `localhost:6379` | Alamat Redis Server |
| `MONGO_URI` | `mongodb://localhost:27017` | URI Koneksi MongoDB |
| `JWT_SECRET_KEY` | `change-me-...` | Secret Key penandatanganan JWT Token |
| `JWT_ACCESS_TOKEN_TTL`| `15m` | Masa berlaku Access Token |
| `JWT_REFRESH_TOKEN_TTL`| `720h` | Masa berlaku Refresh Token |
| `LOG_LEVEL` | `debug` | Level log (`debug`, `info`, `warn`, `error`) |
| `OTEL_ENABLED` | `false` | Mengaktifkan ekspor OpenTelemetry |

---

## 🛠 Manajemen Command (Makefile)

Proyek ini menyediakan berbagai perintah pembantu melalui `Makefile`:

### Build & Run
- `make run` — Menjalankan aplikasi secara langsung (`go run ./cmd/api`)
- `make build` — Mengompilasi binary ke `bin/server`

### Generasi Kode & DI
- `make protogen` — Menggenerasi kode Go pb & OpenAPI dari file `.proto`
- `make wire` — Menggenerasi file `bootstrap/wire_gen.go` menggunakan Google Wire
- `make mocks` — Menggenerasi mock untuk unit testing (`go generate ./internal/...`)

### Database & Migrasi
- `make migrate-up` — Memunculkan seluruh migrasi SQL yang pending
- `make migrate-down` — Mengembalikan (*rollback*) 1 langkah migrasi
- `make migrate-status` — Memeriksa versi migrasi saat ini
- `make migration name=<desc>` — Membuat file migrasi SQL baru

### Testing & Quality
- `make test` — Menjalankan unit test (`./internal/...` `./pkg/...`)
- `make test-cover` — Menjalankan unit test dengan laporan *coverage* HTML (`coverage.html`)
- `make lint` — Menjalankan `golangci-lint`
- `make fmt` — Format kode Go (`gofmt` & `goimports`)

### Docker Operations
- `make docker-up` — Menyalakan seluruh service infra via docker-compose
- `make docker-down` — Mematikan seluruh service infra
- `make docker-logs` — Melihat log aplikasi dari docker-compose

---

## 📡 Spesifikasi API & Endpoints

### 🟢 Probes & Metrics (Public)
| Method | Endpoint | Deskripsi |
| --- | --- | --- |
| `GET` | `/healthz` | Kubernetes Liveness Probe (Standar De-facto) |
| `GET` | `/health` | Alternative Liveness Probe |
| `GET` | `/readyz` | Kubernetes Readiness Probe (Returns 503 jika dependensi down) |
| `GET` | `/metrics` | Prometheus Metrics Endpoint |

### 🔐 Authentication (`/api/v1/auth`)
| Method | Endpoint | Deskripsi | Auth |
| --- | --- | --- | --- |
| `POST` | `/api/v1/auth/login` | Login user & dapatkan sepasang token JWT | Public |
| `POST` | `/api/v1/auth/refresh` | Perbarui Access Token menggunakan Refresh Token | Public |
| `POST` | `/api/v1/auth/logout` | Revoke Refresh Token & Logout | Public |

### 👤 User Management (`/api/v1/users`)
| Method | Endpoint | Deskripsi | Auth |
| --- | --- | --- | --- |
| `POST` | `/api/v1/users` | Membuat user baru (Registrasi) | Public |
| `GET` | `/api/v1/users/{id}` | Mengambil detail user berdasarkan ID | Bearer JWT |
| `PATCH` | `/api/v1/users/{id}` | Memperbarui data user | Bearer JWT |
| `DELETE` | `/api/v1/users/{id}` | Menghapus user | Bearer JWT |

### 📄 Documentation & UI
- **Swagger UI**: Akses melalui peramban di `http://localhost:8080/swagger/`

---

## 🔄 Generasi Kode (Code Generation)

### 1. Protobuf & OpenAPI
Jika Anda mengubah atau menambahkan skema baru di `api/proto/`:
```bash
make protogen
```
*Script `scripts/protogen.sh` (atau `protogen.bat` di Windows) akan secara otomatis menghasilkan struct Go protocol buffer di `gen/pb/` dan file OpenAPI JSON di `gen/openapi/`.*

### 2. Dependency Injection (Wire)
Jika Anda menambah provider baru di `bootstrap/wire.go` atau constructor pada layer `internal/`:
```bash
make wire
```

---

## 🧪 Pengujian & Kualitas Kode

Menjalankan pengujian unit:
```bash
make test
```

Menghasilkan laporan *code coverage*:
```bash
make test-cover
```

Menjalankan linter untuk menjaga kualitas standar kode:
```bash
make lint
```

---

## 📚 Dokumentasi Tambahan

Untuk penjelasan teknis yang mendalam mengenai keputusan desain arsitektur, panduan *clean architecture*, urutan registrasi handler, dan pola penanganan error, silakan merujuk ke:
- [docs/architecture.md](file:///Users/a2375/Projects/go-feature-based-boilerplate/docs/architecture.md)
