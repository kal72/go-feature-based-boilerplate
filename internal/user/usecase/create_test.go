package usecase_test

import (
	"context"
	"testing"

	"go-feature-based-boilerplate/internal/user/dto"
	userErrors "go-feature-based-boilerplate/internal/user/errors"
	"go-feature-based-boilerplate/internal/user/usecase"
	"go-feature-based-boilerplate/internal/user/validator"

	"go-feature-based-boilerplate/internal/user/entity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockRepository is a hand-rolled mock so the test file compiles without mockgen.
type mockRepository struct{ mock.Mock }

func (m *mockRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}
func (m *mockRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}
func (m *mockRepository) Create(ctx context.Context, u *entity.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *mockRepository) Update(ctx context.Context, u *entity.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *mockRepository) Delete(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func TestCreateUsecase_Execute_Success(t *testing.T) {
	repo := new(mockRepository)
	v := validator.New()
	uc := usecase.NewCreateUsecase(repo, v)

	req := dto.CreateUserRequest{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "supersecret",
	}

	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil)

	user, err := uc.Execute(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, req.Name, user.Name)
	assert.Equal(t, req.Email, user.Email)
	repo.AssertExpectations(t)
}

func TestCreateUsecase_Execute_DuplicateEmail(t *testing.T) {
	repo := new(mockRepository)
	v := validator.New()
	uc := usecase.NewCreateUsecase(repo, v)

	req := dto.CreateUserRequest{
		Name:     "Jane Doe",
		Email:    "jane@example.com",
		Password: "supersecret",
	}

	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).
		Return(userErrors.ErrAlreadyExists)

	_, err := uc.Execute(context.Background(), req)

	assert.ErrorIs(t, err, userErrors.ErrAlreadyExists)
	repo.AssertExpectations(t)
}
