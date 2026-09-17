package cryptoutil_test

import (
	"testing"

	"go-feature-based-boilerplate/pkg/cryptoutil"

	"github.com/stretchr/testify/assert"
)

func TestHashSHA256(t *testing.T) {
	input := "super-secret-refresh-token"
	h1 := cryptoutil.HashSHA256(input)
	h2 := cryptoutil.HashSHA256(input)

	assert.NotEmpty(t, h1)
	assert.Equal(t, 64, len(h1)) // 256 bits = 64 hex chars
	assert.Equal(t, h1, h2, "SHA-256 must be deterministic")

	hDiff := cryptoutil.HashSHA256("different-token")
	assert.NotEqual(t, h1, hDiff)
}

func TestPasswordHashing(t *testing.T) {
	pwd := "mypassword123"
	hash, err := cryptoutil.HashPassword(pwd)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	err = cryptoutil.CheckPassword(pwd, hash)
	assert.NoError(t, err)

	err = cryptoutil.CheckPassword("wrongpassword", hash)
	assert.Error(t, err)
}
