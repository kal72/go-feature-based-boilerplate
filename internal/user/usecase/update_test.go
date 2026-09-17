package usecase_test

import (
	"context"
	"testing"

	"go-feature-based-boilerplate/internal/user/dto"
	"go-feature-based-boilerplate/internal/user/entity"
	userErrors "go-feature-based-boilerplate/internal/user/errors"
	"go-feature-based-boilerplate/internal/user/usecase"
	"go-feature-based-boilerplate/internal/user/validator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpdateUsecase_Forbidden_ZeroCallerID(t *testing.T) {
	repo := new(mockRepository)
	v := validator.New()
	uc := usecase.NewUpdateUsecase(repo, v)

	req := dto.UpdateUserRequest{
		ID:       1,
		CallerID: 0, // Unauthenticated or missing caller
		Name:     "Hacker",
	}

	user, err := uc.Execute(context.Background(), req)
	assert.Nil(t, user)
	assert.ErrorIs(t, err, userErrors.ErrForbidden)
}

func TestUpdateUsecase_Forbidden_DifferentUser(t *testing.T) {
	repo := new(mockRepository)
	v := validator.New()
	uc := usecase.NewUpdateUsecase(repo, v)

	req := dto.UpdateUserRequest{
		ID:       1,
		CallerID: 2, // Different user
		Name:     "Hacker",
	}

	user, err := uc.Execute(context.Background(), req)
	assert.Nil(t, user)
	assert.ErrorIs(t, err, userErrors.ErrForbidden)
}

func TestUpdateUsecase_Success(t *testing.T) {
	repo := new(mockRepository)
	v := validator.New()
	uc := usecase.NewUpdateUsecase(repo, v)

	existing := &entity.User{
		ID:   1,
		Name: "Old Name",
	}

	repo.On("FindByID", mock.Anything, uint(1)).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == 1 && u.Name == "New Name"
	})).Return(nil)

	req := dto.UpdateUserRequest{
		ID:       1,
		CallerID: 1, // Same user
		Name:     "New Name",
	}

	updated, err := uc.Execute(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, updated)
	assert.Equal(t, "New Name", updated.Name)
	repo.AssertExpectations(t)
}
