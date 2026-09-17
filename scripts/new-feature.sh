#!/usr/bin/env bash
# ==============================================================================
# Feature Scaffolding Generator
# Creates a clean, standardized Feature-First skeleton according to architecture.
# Usage: ./scripts/new-feature.sh <feature_name>
#    or: make new-feature name=<feature_name>
# ==============================================================================

set -euo pipefail

# ── 1. Validate Input ────────────────────────────────────────────────────────
FEATURE_NAME="${1:-}"

if [ -z "$FEATURE_NAME" ]; then
    echo "❌ Error: Feature name is required."
    echo "Usage: make new-feature name=<feature_name>"
    echo "Example: make new-feature name=product"
    exit 1
fi

# Normalize: trim and lowercase
FEATURE_LOWER=$(echo "$FEATURE_NAME" | tr '[:upper:]' '[:lower:]' | tr '-' '_')

if [[ ! "$FEATURE_LOWER" =~ ^[a-z][a-z0-9_]*$ ]]; then
    echo "❌ Error: Invalid feature name '$FEATURE_NAME'. Must start with a letter and contain only alphanumeric or underscore characters."
    exit 1
fi

TARGET_DIR="internal/${FEATURE_LOWER}"
PROTO_DIR="api/proto/${FEATURE_LOWER}"

if [ -d "$TARGET_DIR" ]; then
    echo "❌ Error: Feature '${FEATURE_LOWER}' already exists at ${TARGET_DIR}."
    exit 1
fi

# Detect module name from go.mod
MODULE=$(grep '^module ' go.mod 2>/dev/null | awk '{print $2}' || echo "go-feature-based-boilerplate")

# Calculate PascalCase and camelCase
PASCAL_NAME=$(echo "$FEATURE_LOWER" | awk -F'_' '{for(i=1;i<=NF;i++) printf toupper(substr($i,1,1)) substr($i,2); print ""}')
CAMEL_NAME=$(echo "$PASCAL_NAME" | awk '{print tolower(substr($0,1,1)) substr($0,2)}')

# Ensure entity file doesn't collide with Go test naming convention (*_test.go)
ENTITY_FILE="${FEATURE_LOWER}.go"
if [[ "$FEATURE_LOWER" == *"_test" ]] || [[ "$FEATURE_LOWER" == "test" ]]; then
    ENTITY_FILE="${FEATURE_LOWER}_entity.go"
fi

echo "🚀 Generating new feature: '${FEATURE_LOWER}' (Struct: ${PASCAL_NAME})..."

# ── 2. Create Directory Structure ────────────────────────────────────────────
mkdir -p "${TARGET_DIR}/dto"
mkdir -p "${TARGET_DIR}/entity"
mkdir -p "${TARGET_DIR}/errors"
mkdir -p "${TARGET_DIR}/handler"
mkdir -p "${TARGET_DIR}/repository/postgres"
mkdir -p "${TARGET_DIR}/usecase"
mkdir -p "${TARGET_DIR}/validator"
mkdir -p "${PROTO_DIR}"

# ── 3. Generate Proto Contract ───────────────────────────────────────────────
cat <<EOF > "${PROTO_DIR}/${FEATURE_LOWER}.proto"
syntax = "proto3";

package ${FEATURE_LOWER}.v1;

option go_package = "${MODULE}/gen/pb/${FEATURE_LOWER};${FEATURE_LOWER}pb";

import "google/api/annotations.proto";

service ${PASCAL_NAME}Service {
  rpc Create${PASCAL_NAME}(Create${PASCAL_NAME}Request) returns (Create${PASCAL_NAME}Response) {
    option (google.api.http) = {
      post: "/api/v1/${FEATURE_LOWER}s"
      body: "*"
    };
  }

  rpc Get${PASCAL_NAME}(Get${PASCAL_NAME}Request) returns (Get${PASCAL_NAME}Response) {
    option (google.api.http) = {
      get: "/api/v1/${FEATURE_LOWER}s/{id}"
    };
  }
}

message ${PASCAL_NAME}Proto {
  uint64 id = 1;
  string name = 2;
  string created_at = 3;
  string updated_at = 4;
}

message Create${PASCAL_NAME}Request {
  string name = 1;
}

message Create${PASCAL_NAME}Response {
  ${PASCAL_NAME}Proto ${FEATURE_LOWER} = 1;
}

message Get${PASCAL_NAME}Request {
  uint64 id = 1;
}

message Get${PASCAL_NAME}Response {
  ${PASCAL_NAME}Proto ${FEATURE_LOWER} = 1;
}
EOF

# ── 4. Generate Entity ───────────────────────────────────────────────────────
cat <<EOF > "${TARGET_DIR}/entity/${ENTITY_FILE}"
package entity

import "time"

// ${PASCAL_NAME} represents the ${FEATURE_LOWER} domain entity and database model.
type ${PASCAL_NAME} struct {
	ID        uint      \`gorm:"primaryKey;autoIncrement" json:"id"\`
	Name      string    \`gorm:"size:255;not null" json:"name"\`
	CreatedAt time.Time \`gorm:"autoCreateTime" json:"created_at"\`
	UpdatedAt time.Time \`gorm:"autoUpdateTime" json:"updated_at"\`
}

func (${PASCAL_NAME}) TableName() string {
	return "${FEATURE_LOWER}s"
}
EOF

# ── 5. Generate DTO ──────────────────────────────────────────────────────────
cat <<EOF > "${TARGET_DIR}/dto/request.go"
package dto

// Create${PASCAL_NAME}Request holds inputs for creating a new ${FEATURE_LOWER}.
type Create${PASCAL_NAMERequest:-Create${PASCAL_NAME}Request} struct {
	Name string \`json:"name" validate:"required,min=2,max=255"\`
}
EOF

# Clean up DTO request struct name
sed -i '' "s/Create${PASCAL_NAMERequest:-Create${PASCAL_NAME}Request}/Create${PASCAL_NAME}Request/g" "${TARGET_DIR}/dto/request.go" 2>/dev/null || true

cat <<EOF > "${TARGET_DIR}/dto/response.go"
package dto

import "time"

// ${PASCAL_NAME}Response represents the domain response data for a ${FEATURE_LOWER}.
type ${PASCAL_NAME}Response struct {
	ID        uint      \`json:"id"\`
	Name      string    \`json:"name"\`
	CreatedAt time.Time \`json:"created_at"\`
	UpdatedAt time.Time \`json:"updated_at"\`
}
EOF

# ── 6. Generate Domain Errors ────────────────────────────────────────────────
cat <<EOF > "${TARGET_DIR}/errors/errors.go"
package errors

import (
	"${MODULE}/pkg/errorutil"
)

var (
	ErrNotFound      = errorutil.New(errorutil.CodeNotFound, "${FEATURE_LOWER} not found")
	ErrAlreadyExists = errorutil.New(errorutil.CodeAlreadyExists, "${FEATURE_LOWER} already exists")
	ErrInvalidInput  = errorutil.New(errorutil.CodeInvalidInput, "invalid ${FEATURE_LOWER} input")
)
EOF

# ── 7. Generate Validator ────────────────────────────────────────────────────
cat <<EOF > "${TARGET_DIR}/validator/validator.go"
package validator

import (
	"${MODULE}/pkg/errorutil"
	pkgvalidator "${MODULE}/pkg/validator"
)

// Validator wraps the shared validator for ${FEATURE_LOWER}.
type Validator struct{}

// New creates a new ${PASCAL_NAME} validator instance.
func New() *Validator {
	return &Validator{}
}

// Validate validates the given struct and returns an AppError on failure.
func (v *Validator) Validate(s any) error {
	if err := pkgvalidator.ValidateStruct(s); err != nil {
		return errorutil.Wrap(errorutil.CodeInvalidInput, "validation failed", err)
	}
	return nil
}
EOF

# ── 8. Generate Repository Interface & Postgres Adapter ─────────────────────
cat <<EOF > "${TARGET_DIR}/repository/interface.go"
package repository

import (
	"context"

	"${MODULE}/internal/${FEATURE_LOWER}/entity"
)

// Repository defines data access operations for ${PASCAL_NAME}.
type Repository interface {
	FindByID(ctx context.Context, id uint) (*entity.${PASCAL_NAME}, error)
	Create(ctx context.Context, item *entity.${PASCAL_NAME}) error
	Update(ctx context.Context, item *entity.${PASCAL_NAME}) error
	Delete(ctx context.Context, id uint) error
}
EOF

cat <<EOF > "${TARGET_DIR}/repository/postgres/repository.go"
package postgres

import (
	"context"
	"errors"

	infraPostgres "${MODULE}/infrastructure/database/postgres"
	"${MODULE}/internal/${FEATURE_LOWER}/entity"
	${FEATURE_LOWER}Errors "${MODULE}/internal/${FEATURE_LOWER}/errors"
	"${MODULE}/pkg/errorutil"

	"gorm.io/gorm"
)

// ${PASCAL_NAME}Repository implements repository.Repository using PostgreSQL via GORM.
type ${PASCAL_NAME}Repository struct {
	db *gorm.DB
}

// New${PASCAL_NAME}Repository creates a new PostgreSQL-backed ${FEATURE_LOWER} repository.
func New${PASCAL_NAME}Repository(db *gorm.DB) *${PASCAL_NAME}Repository {
	return &${PASCAL_NAME}Repository{db: db}
}

func (r *${PASCAL_NAME}Repository) getDB(ctx context.Context) *gorm.DB {
	return infraPostgres.GetDB(ctx, r.db)
}

func (r *${PASCAL_NAME}Repository) FindByID(ctx context.Context, id uint) (*entity.${PASCAL_NAME}, error) {
	var item entity.${PASCAL_NAME}
	if err := r.getDB(ctx).First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ${FEATURE_LOWER}Errors.ErrNotFound
		}
		return nil, errorutil.Wrap(errorutil.CodeInternal, "find ${FEATURE_LOWER} by id", err)
	}
	return &item, nil
}

func (r *${PASCAL_NAME}Repository) Create(ctx context.Context, item *entity.${PASCAL_NAME}) error {
	if err := r.getDB(ctx).Create(item).Error; err != nil {
		return errorutil.Wrap(errorutil.CodeInternal, "create ${FEATURE_LOWER}", err)
	}
	return nil
}

func (r *${PASCAL_NAME}Repository) Update(ctx context.Context, item *entity.${PASCAL_NAME}) error {
	result := r.getDB(ctx).Save(item)
	if result.Error != nil {
		return errorutil.Wrap(errorutil.CodeInternal, "update ${FEATURE_LOWER}", result.Error)
	}
	if result.RowsAffected == 0 {
		return ${FEATURE_LOWER}Errors.ErrNotFound
	}
	return nil
}

func (r *${PASCAL_NAME}Repository) Delete(ctx context.Context, id uint) error {
	result := r.getDB(ctx).Delete(&entity.${PASCAL_NAME}{}, id)
	if result.Error != nil {
		return errorutil.Wrap(errorutil.CodeInternal, "delete ${FEATURE_LOWER}", result.Error)
	}
	if result.RowsAffected == 0 {
		return ${FEATURE_LOWER}Errors.ErrNotFound
	}
	return nil
}
EOF

# ── 9. Generate Usecases & Unit Tests ────────────────────────────────────────
cat <<EOF > "${TARGET_DIR}/usecase/create.go"
package usecase

import (
	"context"

	"${MODULE}/internal/${FEATURE_LOWER}/dto"
	"${MODULE}/internal/${FEATURE_LOWER}/entity"
	"${MODULE}/internal/${FEATURE_LOWER}/repository"
	"${MODULE}/internal/${FEATURE_LOWER}/validator"
)

// CreateUsecase handles creation of a new ${FEATURE_LOWER}.
type CreateUsecase struct {
	repo      repository.Repository
	validator *validator.Validator
}

// NewCreateUsecase constructs a CreateUsecase.
func NewCreateUsecase(repo repository.Repository, v *validator.Validator) *CreateUsecase {
	return &CreateUsecase{repo: repo, validator: v}
}

func (u *CreateUsecase) Execute(ctx context.Context, req dto.Create${PASCAL_NAME}Request) (*entity.${PASCAL_NAME}, error) {
	if err := u.validator.Validate(&req); err != nil {
		return nil, err
	}

	item := &entity.${PASCAL_NAME}{
		Name: req.Name,
	}

	if err := u.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}
EOF

cat <<EOF > "${TARGET_DIR}/usecase/find.go"
package usecase

import (
	"context"

	"${MODULE}/internal/${FEATURE_LOWER}/entity"
	"${MODULE}/internal/${FEATURE_LOWER}/repository"
)

// FindUsecase handles retrieval of ${FEATURE_LOWER} records.
type FindUsecase struct {
	repo repository.Repository
}

// NewFindUsecase constructs a FindUsecase.
func NewFindUsecase(repo repository.Repository) *FindUsecase {
	return &FindUsecase{repo: repo}
}

func (u *FindUsecase) FindByID(ctx context.Context, id uint) (*entity.${PASCAL_NAME}, error) {
	return u.repo.FindByID(ctx, id)
}
EOF

cat <<EOF > "${TARGET_DIR}/usecase/create_test.go"
package usecase_test

import (
	"context"
	"testing"

	"${MODULE}/internal/${FEATURE_LOWER}/dto"
	"${MODULE}/internal/${FEATURE_LOWER}/entity"
	"${MODULE}/internal/${FEATURE_LOWER}/usecase"
	"${MODULE}/internal/${FEATURE_LOWER}/validator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type Mock${PASCAL_NAME}Repository struct {
	mock.Mock
}

func (m *Mock${PASCAL_NAME}Repository) FindByID(ctx context.Context, id uint) (*entity.${PASCAL_NAME}, error) {
	args := m.Called(ctx, id)
	if item := args.Get(0); item != nil {
		return item.(*entity.${PASCAL_NAME}), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *Mock${PASCAL_NAME}Repository) Create(ctx context.Context, item *entity.${PASCAL_NAME}) error {
	return m.Called(ctx, item).Error(0)
}

func (m *Mock${PASCAL_NAME}Repository) Update(ctx context.Context, item *entity.${PASCAL_NAME}) error {
	return m.Called(ctx, item).Error(0)
}

func (m *Mock${PASCAL_NAME}Repository) Delete(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func TestCreateUsecase_Success(t *testing.T) {
	mockRepo := new(Mock${PASCAL_NAME}Repository)
	v := validator.New()
	uc := usecase.NewCreateUsecase(mockRepo, v)

	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(item *entity.${PASCAL_NAME}) bool {
		return item.Name == "Sample Name"
	})).Return(nil)

	res, err := uc.Execute(context.Background(), dto.Create${PASCAL_NAME}Request{
		Name: "Sample Name",
	})

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "Sample Name", res.Name)
	mockRepo.AssertExpectations(t)
}

func TestCreateUsecase_ValidationError(t *testing.T) {
	mockRepo := new(Mock${PASCAL_NAME}Repository)
	v := validator.New()
	uc := usecase.NewCreateUsecase(mockRepo, v)

	// Name too short (min=2 in validation tag)
	res, err := uc.Execute(context.Background(), dto.Create${PASCAL_NAME}Request{
		Name: "A",
	})

	assert.Error(t, err)
	assert.Nil(t, res)
}
EOF

# ── 10. Generate Handler ─────────────────────────────────────────────────────
cat <<EOF > "${TARGET_DIR}/handler/handler.go"
package handler

import (
	"${MODULE}/internal/${FEATURE_LOWER}/usecase"
	"${MODULE}/internal/${FEATURE_LOWER}/validator"
)

// Handler serves ${FEATURE_LOWER} gRPC requests.
type Handler struct {
	createUsecase *usecase.CreateUsecase
	findUsecase   *usecase.FindUsecase
	validator     *validator.Validator
}

// NewHandler constructs a ${PASCAL_NAME} gRPC handler.
func NewHandler(
	createUsecase *usecase.CreateUsecase,
	findUsecase *usecase.FindUsecase,
	validator *validator.Validator,
) *Handler {
	return &Handler{
		createUsecase: createUsecase,
		findUsecase:   findUsecase,
		validator:     validator,
	}
}
EOF

# ── 11. Generate ProviderSet (Google Wire) ───────────────────────────────────
cat <<EOF > "${TARGET_DIR}/provider.go"
package ${FEATURE_LOWER}

import (
	"${MODULE}/internal/${FEATURE_LOWER}/handler"
	"${MODULE}/internal/${FEATURE_LOWER}/repository"
	"${MODULE}/internal/${FEATURE_LOWER}/repository/postgres"
	"${MODULE}/internal/${FEATURE_LOWER}/usecase"
	"${MODULE}/internal/${FEATURE_LOWER}/validator"

	"github.com/google/wire"
)

// ProviderSet wires all ${FEATURE_LOWER} feature components.
var ProviderSet = wire.NewSet(
	validator.New,
	usecase.NewCreateUsecase,
	usecase.NewFindUsecase,
	handler.NewHandler,
	postgres.New${PASCAL_NAME}Repository,
	wire.Bind(new(repository.Repository), new(*postgres.${PASCAL_NAME}Repository)),
)
EOF

echo ""
echo "✨ Feature '${FEATURE_LOWER}' successfully created!"
echo ""
echo "📁 Generated Files:"
echo "   ├── api/proto/${FEATURE_LOWER}/${FEATURE_LOWER}.proto"
echo "   └── internal/${FEATURE_LOWER}/"
echo "       ├── dto/{request.go, response.go}"
echo "       ├── entity/${ENTITY_FILE}"
echo "       ├── errors/errors.go"
echo "       ├── handler/handler.go"
echo "       ├── repository/{interface.go, postgres/repository.go}"
echo "       ├── usecase/{create.go, create_test.go, find.go}"
echo "       ├── validator/validator.go"
echo "       └── provider.go"
echo ""
echo "🚀 Next Steps:"
echo "   1. Review proto file: api/proto/${FEATURE_LOWER}/${FEATURE_LOWER}.proto"
echo "   2. Generate protobuf stubs: make protogen"
echo "   3. Add ${FEATURE_LOWER}.ProviderSet to bootstrap/wire.go"
echo "   4. Regenerate Wire graph:   make wire"
echo "   5. Register gRPC & Gateway handlers in bootstrap/grpc.go & gateway.go"
echo ""
