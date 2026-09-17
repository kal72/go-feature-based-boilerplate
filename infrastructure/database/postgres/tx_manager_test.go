package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTransactionManager_GetDB_And_TxFromContext(t *testing.T) {
	defaultDB := &gorm.DB{Config: &gorm.Config{}, Statement: &gorm.Statement{}}
	txDB := &gorm.DB{Config: &gorm.Config{}, Statement: &gorm.Statement{}}

	ctx := context.Background()

	// 1. Without transaction in context
	gotDB := GetDB(ctx, defaultDB)
	assert.NotNil(t, gotDB)
	extractedTx, ok := TxFromContext(ctx)
	assert.False(t, ok)
	assert.Nil(t, extractedTx)

	// 2. With transaction in context
	ctxWithTx := context.WithValue(ctx, txContextKey{}, txDB)
	gotTxDB := GetDB(ctxWithTx, defaultDB)
	assert.NotNil(t, gotTxDB)
	assert.Equal(t, txDB, gotTxDB)

	extractedTx, ok = TxFromContext(ctxWithTx)
	assert.True(t, ok)
	assert.Equal(t, txDB, extractedTx)
}

func TestTransactionManager_NestedRunInTransaction(t *testing.T) {
	dummyTx := &gorm.DB{}
	tm := NewTransactionManager(nil)

	// When already inside a transaction, RunInTransaction should reuse context without calling tm.db
	ctxWithTx := context.WithValue(context.Background(), txContextKey{}, dummyTx)

	called := false
	err := tm.RunInTransaction(ctxWithTx, func(ctx context.Context) error {
		called = true
		// Verify the context passed still has the tx
		tx, ok := TxFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, dummyTx, tx)
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)

	// Error propagation in nested call
	expectedErr := errors.New("nested rollback error")
	err = tm.RunInTransaction(ctxWithTx, func(ctx context.Context) error {
		return expectedErr
	})
	assert.Equal(t, expectedErr, err)
}
