package contextutil_test

import (
	"context"
	"testing"

	"go-feature-based-boilerplate/pkg/contextutil"

	"github.com/stretchr/testify/assert"
)

func TestUserIDContext(t *testing.T) {
	ctx := context.Background()

	_, ok := contextutil.GetUserID(ctx)
	assert.False(t, ok)

	ctxWithUser := contextutil.WithUserID(ctx, 42)
	id, ok := contextutil.GetUserID(ctxWithUser)
	assert.True(t, ok)
	assert.Equal(t, uint(42), id)
}

func TestRoleContext(t *testing.T) {
	ctx := context.Background()

	_, ok := contextutil.GetRole(ctx)
	assert.False(t, ok)

	ctxWithRole := contextutil.WithRole(ctx, "admin")
	role, ok := contextutil.GetRole(ctxWithRole)
	assert.True(t, ok)
	assert.Equal(t, "admin", role)
}
