package usecase_test

import (
	"context"
	"testing"

	userErrors "go-feature-based-boilerplate/internal/user/errors"
	"go-feature-based-boilerplate/internal/user/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteUsecase_Forbidden_ZeroCallerID(t *testing.T) {
	repo := new(mockRepository)
	uc := usecase.NewDeleteUsecase(repo)

	err := uc.Execute(context.Background(), 1, 0) // target 1, caller 0
	assert.ErrorIs(t, err, userErrors.ErrForbidden)
}

func TestDeleteUsecase_Forbidden_DifferentUser(t *testing.T) {
	repo := new(mockRepository)
	uc := usecase.NewDeleteUsecase(repo)

	err := uc.Execute(context.Background(), 1, 2) // target 1, caller 2
	assert.ErrorIs(t, err, userErrors.ErrForbidden)
}

func TestDeleteUsecase_Success(t *testing.T) {
	repo := new(mockRepository)
	uc := usecase.NewDeleteUsecase(repo)

	repo.On("Delete", mock.Anything, uint(1)).Return(nil)

	err := uc.Execute(context.Background(), 1, 1) // target 1, caller 1
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
