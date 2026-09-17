# AGENTS.md — AI Agent Guidance & Architectural Contract

Panduan ini mendefinisikan aturan, konvensi arsitektur, batasan ketat (*guardrails*), dan alur kerja (*workflows*) bagi AI Agent yang berkontribusi dalam pengembangan dan pemeliharaan repositori **Go Feature-Based Modular Clean Architecture Boilerplate**.

---

## 1. Ikhtisar Proyek & Arsitektur Utama

* **Arsitektur**: Feature-Based Modular Clean Architecture yang menggabungkan prinsip Hexagonal Architecture (Ports & Adapters) dan pendekatan Lightweight Domain-Driven Design (DDD).
* **Dual Protocol**:
  * **gRPC Server** (port `:50051`) sebagai transport protokol utama berkinerja tinggi.
  * **HTTP REST Gateway** (port `:8080`) menggunakan `grpc-gateway` yang memetakan route HTTP ke gRPC secara otomatis.
  * **Native Swagger UI** disajikan di `/swagger/`.
* **Tech Stack**:
  * **Bahasa**: Go (`go.mod` menargetkan Go versi modern).
  * **Database**: PostgreSQL (GORM) dengan skema migrasi terkelola (`golang-migrate`).
  * **Dependency Injection**: Google Wire (compile-time type-safe DI).
  * **Observability (Full-Stack APM)**: Structured Logging (**Uber Zap** via `pkg/logger` 100% JSON format), OpenTelemetry Tracing (HTTP Ingress via `otelhttp`, gRPC via `otelgrpc`, Database via GORM `otel`, Redis via `redisotel`, Outbound HTTP via `pkg/httpclient`), Prometheus Metrics (`/metrics`), serta Continuous Profiling (`pprof` di `/debug/pprof/`).
  * **Extensions**: Redis client (`infrastructure/cache/redis`) dan MongoDB driver (`infrastructure/database/mongo`).

---

## 2. Struktur Direktori & Tanggung Jawab Layer

```text
.
├── api/proto/             # Definisi kontrak gRPC Protocol Buffer (.proto)
├── bootstrap/             # Composition Root (Wire DI, inisialisasi Server gRPC, Gateway, App)
├── cmd/api/               # Minimal application entrypoint (main.go saja)
├── deployments/           # Dockerfile & docker-compose.yml
├── docs/                  # Dokumentasi arsitektur detail (architecture.md)
├── gen/                   # Generated code (pb/ dan openapi/). DILARANG edit manual!
├── infrastructure/        # Implementasi driver eksternal, DB, config, middleware, telemetry
├── internal/              # Fitur bisnis (Feature-First modular domains)
│   ├── <feature>/
│   │   ├── dto/           # Data Transfer Objects (transport <-> usecase)
│   │   ├── entity/        # Core domain model & business rules murni
│   │   ├── errors/        # Typed business errors per domain
│   │   ├── handler/       # gRPC server handlers
│   │   ├── repository/    # Repository interfaces (ports) & adapters (postgres, dsb.)
│   │   ├── usecase/       # Application & business workflow logic
│   │   ├── validator/     # Input & domain validation logic
│   │   └── provider.go    # Wire ProviderSet untuk feature ini
├── migrations/            # SQL migration files (*.up.sql, *.down.sql)
├── pkg/                   # Generic utility packages (independen, zero external business logic)
│   ├── contextutil/       # Context helpers (request_id, user_id, role)
│   ├── cryptoutil/        # Password hashing (bcrypt) & encryption
│   ├── httpclient/        # Resilient outbound HTTP client (pooling, retry, trace/request_id injection)
│   ├── idempotency/       # Idempotency execution engine (Memory & Redis storage)
│   ├── jwtutil/           # JWT generation & token validation
│   ├── logger/            # Enterprise microservice JSON logger (Uber Zap wrapper)
│   ├── pagination/        # Offset & Cursor-based pagination with SQL sanitizer
│   ├── routine/           # Concurrency toolkit (Worker Pool, Retry, Singleflight)
│   └── validator/         # Custom validation tags & rules
└── scripts/               # Utility scripts (protogen.sh, protogen.bat)
```

---

## 3. Batasan Ketat & Aturan Dependensi (Strict Guardrails)

AI Agent **WAJIB** mematuhi aturan ketergantungan layer berikut:

### Rule 1: Arah Ketergantungan (Dependency Inversion)
* Layer dalam **DILARANG** mengimpor layer luar:
  * `entity` adalah domain murni: **DILARANG** mengimpor `usecase`, `handler`, `repository`, `infrastructure`, atau framework eksternal seperti GORM/gRPC.
  * `usecase` hanya boleh mengimpor `entity`, `dto`, `validator`, dan `repository` (interface/port). **DILARANG** mengimpor driver database spesifik (GORM, pgx) atau gRPC transport.
  * `repository` mengimplementasikan interface yang dibutuhkan `usecase`.
  * `handler` menerjemahkan request gRPC/HTTP menjadi DTO dan memanggil `usecase`.

### Rule 2: Isolasi Antar-Fitur (`internal/<feature>`)
* Satu fitur di `internal/<featureA>` **DILARANG** mengimpor implementasi konkret dari `internal/<featureB>`.
* Komunikasi lintas-fitur hanya boleh dilakukan melalui **Exported Provider / Interface** yang didefinisikan pada level root package fitur tersebut (contoh: `auth.UserProvider` yang diimplementasikan di `user`).

### Rule 3: Kemurnian `pkg/`
* Seluruh paket di bawah `pkg/` harus bersifat **generic** dan **reusable**:
  * **DILARANG** mengimpor paket dari `internal/` atau `infrastructure/`.
  * `pkg/` dapat digunakan oleh `internal`, `infrastructure`, atau `bootstrap`.

### Rule 4: Dilarang Mengubah File Tergenerasi (`gen/`) Secara Manual
* File di `gen/pb/` dan `gen/openapi/` dihasilkan otomatis oleh compiler `protoc`.
* Jika butuh mengubah kontrak data:
  1. Ubah file `.proto` di `api/proto/<feature>/v1/`.
  2. Jalankan `make protogen` (atau `./scripts/protogen.sh`).

### Rule 5: Dilarang Meletakkan Business Logic di `cmd/` atau `bootstrap/`
* `cmd/api/main.go` hanya boleh memanggil `bootstrap.InitializeApp()`.
* `bootstrap/` hanya bertindak sebagai *Composition Root* untuk perakitan dependensi (Wire) dan server lifecycle runner.

---

## 4. Standar Kode & Konvensi Teknis

### A. Context Propagation
* `ctx context.Context` harus selalu menjadi parameter **pertama** pada setiap method di `handler`, `usecase`, `repository`, dan `pkg/logger`.
* Jangan gunakan `context.Background()` di dalam business flow; selalu teruskan context dari caller.

### B. Standard Logging (`pkg/logger`)
* Selalu gunakan `pkg/logger.Logger` untuk mencatat log aplikasi di layer usecase/handler:
  ```go
  // BENAR: Bersih tanpa fields ...Field
  logger.Info(ctx, "user registered successfully")
  logger.Error(ctx, "database query failed", err)

  // BENAR: Data custom terstruktur melalui method fluent
  logger.WithData(responseObject).Info(ctx, "payment response received")
  logger.With("order_id", orderID).Warn(ctx, "order cancelled by timeout")
  ```
* **DILARANG** menambahkan parameter variadic `(fields ...Field)` pada method logging standar. Seluruh field korelasi (`trace_id`, `span_id`, `request_id`, `user_id`, `role`, `rpc.*`, `service`, `env`, `host`) diekstrak otomatis dari `ctx`.
* Log format selalu **100% Structured JSON**.
* **Audit Payload Policy (gRPC Interceptor)**:
  * `request`: Selalu dicatat (dengan masking field sensitif seperti `password`, `token`, `secret`, `credit_card`).
  * `response`: **HANYA dicatat ketika status code bukan OK (`code != codes.OK`)** demi menjaga efisiensi storage log dan CPU di production.

### C. Error Handling
* Jangan pernah mengabaikan error (`_ = err` dilarang kecuali pada operasi close yang sudah dijamin aman).
* Gunakan typed error di `internal/<feature>/errors/`.
* Pada gRPC handler, terjemahkan domain error ke gRPC status code yang semantik:
  * `codes.InvalidArgument` (400) untuk validasi gagal.
  * `codes.Unauthenticated` (401) untuk token tidak valid atau missing.
  * `codes.PermissionDenied` (403) untuk akses dilarang.
  * `codes.NotFound` (404) untuk data tidak ditemukan.
  * `codes.AlreadyExists` (409) untuk konflik duplikasi (email/username unik).
  * `codes.DeadlineExceeded` (504) untuk batas waktu request tercapai (`TimeoutInterceptor`).
  * `codes.Canceled` (499) untuk request yang dibatalkan oleh klien.
  * `codes.Internal` (500) untuk kegagalan sistem internal. Error internal harus disanitasi agar tidak membocorkan informasi query/credential ke client.

### D. Keamanan Data Sensitif (PII / Masking)
* Dilarang mencatat plain-text password, auth token, PIN, OTP, CVV, atau nomor kartu kredit ke file log atau respons error.
* Gunakan sanitizer yang mengubah data sensitif menjadi `[REDACTED]`.

### E. Dokumentasi Kode (Go Doc Comments)
* Setiap struct, interface, function, method, dan konstan yang diekspos (*exported*, diawali huruf kapital) **WAJIB** memiliki komentar dokumentasi Go standard dalam format:
  ```go
  // MethodName describes what this method accomplishes and any notable behaviors.
  func (r *Repo) MethodName(ctx context.Context, ...) (...)
  ```

---

## 5. Panduan Menambahkan Fitur Baru (Step-by-Step Recipe)

Ketika diminta mengimplementasikan fitur baru (misal: `order`), AI Agent harus mengikuti urutan berikut:

1. **Definisi Kontrak Protobuf**:
   * Buat file proto di `api/proto/order/v1/order.proto`.
   * Definisikan message request, message response, dan service gRPC lengkap dengan opsi `google.api.http` untuk mapping REST gateway.
   * Jalankan `make protogen`.
2. **Database Migration** (jika memerlukan tabel baru):
   * Buat migrasi SQL baru di `migrations/`: `XXXXXX_create_orders_table.up.sql` dan `XXXXXX_create_orders_table.down.sql`.
3. **Domain Entity & Errors**:
   * Buat `internal/order/entity/order.go` (definisi domain model murni).
   * Buat `internal/order/errors/errors.go` (sentinel / typed domain errors).
4. **Data Access (Repository)**:
   * Definisikan interface `OrderRepository` di `internal/order/repository/repository.go`.
   * Implementasikan repository PostgreSQL di `internal/order/repository/postgres/order_repository.go`.
5. **Business Logic (Usecase & DTO)**:
   * Buat DTO di `internal/order/dto/`.
   * Implementasikan usecase di `internal/order/usecase/`.
   * Tambahkan validator di `internal/order/validator/`.
6. **Transport Layer (gRPC Handler)**:
   * Implementasikan server gRPC di `internal/order/handler/handler.go` yang mengimplementasikan interface hasil generasi proto.
7. **Wire Dependency Injection**:
   * Buat `internal/order/provider.go` yang mengemas `wire.NewSet(...)`.
   * Daftarkan ProviderSet fitur baru di `bootstrap/wire.go`.
   * Daftarkan service registration di `bootstrap/grpc.go` dan `bootstrap/gateway.go`.
   * Jalankan `make wire` atau sinkronkan `bootstrap/wire_gen.go`.
8. **Unit & Integration Testing**:
   * Tulis unit test untuk usecase dan repository mock (`*_test.go`).
   * Pastikan seluruh test lolos dengan flag `-race`.

---

## 6. Checklist Verifikasi AI Agent

Sebelum menyelesaikan tugas atau menyerahkan perubahan ke pengguna, AI Agent **WAJIB** menjalankan checklist pengujian berikut di terminal:

```bash
# 1. Format kode Go
gofmt -w -s .

# 2. Jalankan seluruh unit test dengan data race detector
go test -race ./...

# 3. Verifikasi kompilasi binary tanpa CGO
make build
```

Jika salah satu tahap di atas gagal, perbaiki masalah tersebut terlebih dahulu sebelum menandai tugas selesai.
