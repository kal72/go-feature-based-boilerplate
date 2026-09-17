package jwtutil

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when the JWT string cannot be parsed or signature verification fails.
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when the token's exp timestamp has passed.
	ErrExpiredToken = errors.New("token has expired")
	// ErrInvalidSubject is returned when the token subject is empty or not a valid positive integer.
	ErrInvalidSubject = errors.New("invalid token subject")
)

// Claims represents the standard and custom claims contained in application JWTs.
type Claims struct {
	jwt.RegisteredClaims
	UserID uint   `json:"user_id,omitempty"`
	Role   string `json:"role,omitempty"`
}

// GenerateToken creates and signs a new JWT access token using HMAC-SHA256.
func GenerateToken(userID uint, role string, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	subStr := strconv.FormatUint(uint64(userID), 10)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subStr,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		UserID: userID,
		Role:   role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("jwtutil: sign token: %w", err)
	}
	return signed, nil
}

// ValidateToken parses, validates the signature, and verifies the claims of a JWT string.
func ValidateToken(tokenStr string, secret []byte) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// Ensure UserID is populated; if empty in custom claim, populate from Subject
	if claims.UserID == 0 && claims.Subject != "" {
		id, parseErr := strconv.ParseUint(claims.Subject, 10, 64)
		if parseErr == nil && id > 0 {
			claims.UserID = uint(id)
		}
	}

	if claims.UserID == 0 {
		return nil, ErrInvalidSubject
	}

	return claims, nil
}
