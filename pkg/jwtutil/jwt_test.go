package jwtutil_test

import (
	"strconv"
	"testing"
	"time"

	"go-feature-based-boilerplate/pkg/jwtutil"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testSecret = []byte("super-secret-key-1234567890123456")

func TestGenerateAndValidateToken_Success(t *testing.T) {
	userID := uint(42)
	role := "admin"
	ttl := 15 * time.Minute

	tokenStr, err := jwtutil.GenerateToken(userID, role, testSecret, ttl)
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)

	claims, err := jwtutil.ValidateToken(tokenStr, testSecret)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "42", claims.Subject)
	assert.Equal(t, role, claims.Role)
	assert.True(t, claims.ExpiresAt.After(time.Now()))
}

func TestValidateToken_Expired(t *testing.T) {
	// Generate token that was expired 1 minute ago
	tokenStr, err := jwtutil.GenerateToken(10, "user", testSecret, -1*time.Minute)
	require.NoError(t, err)

	claims, err := jwtutil.ValidateToken(tokenStr, testSecret)
	assert.ErrorIs(t, err, jwtutil.ErrExpiredToken)
	assert.Nil(t, claims)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	tokenStr, err := jwtutil.GenerateToken(10, "user", testSecret, 15*time.Minute)
	require.NoError(t, err)

	wrongSecret := []byte("completely-different-secret-key!")
	claims, err := jwtutil.ValidateToken(tokenStr, wrongSecret)
	assert.ErrorIs(t, err, jwtutil.ErrInvalidToken)
	assert.Nil(t, claims)
}

func TestValidateToken_Malformed(t *testing.T) {
	claims, err := jwtutil.ValidateToken("not.a.valid.jwt.token", testSecret)
	assert.ErrorIs(t, err, jwtutil.ErrInvalidToken)
	assert.Nil(t, claims)
}

func TestValidateToken_InvalidSubject(t *testing.T) {
	// Token with subject "0"
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "0",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
	})
	tokenStr, err := token.SignedString(testSecret)
	require.NoError(t, err)

	claims, err := jwtutil.ValidateToken(tokenStr, testSecret)
	assert.ErrorIs(t, err, jwtutil.ErrInvalidSubject)
	assert.Nil(t, claims)

	// Token with non-numeric subject
	tokenBadSub := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "abc",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
	})
	tokenStrBad, err := tokenBadSub.SignedString(testSecret)
	require.NoError(t, err)

	claims, err = jwtutil.ValidateToken(tokenStrBad, testSecret)
	assert.ErrorIs(t, err, jwtutil.ErrInvalidSubject)
	assert.Nil(t, claims)
}

func TestValidateToken_BackwardCompatibleWithRegisteredClaims(t *testing.T) {
	// Old token that only has Subject and no custom UserID field
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(99, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
	})
	tokenStr, err := token.SignedString(testSecret)
	require.NoError(t, err)

	claims, err := jwtutil.ValidateToken(tokenStr, testSecret)
	require.NoError(t, err)
	assert.Equal(t, uint(99), claims.UserID)
	assert.Equal(t, "99", claims.Subject)
}
