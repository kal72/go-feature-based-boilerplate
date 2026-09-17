package entity

import "time"

// RefreshToken represents a persisted refresh token for a user session.
type RefreshToken struct {
	ID        uint      `gorm:"primarykey"`
	UserID    uint      `gorm:"not null;index"`
	Token     string    `gorm:"uniqueIndex;not null"`
	Revoked   bool      `gorm:"not null;default:false"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// TableName tells GORM which table to use.
func (RefreshToken) TableName() string { return "refresh_tokens" }

// IsExpired reports whether the token has passed its expiry time.
func (t *RefreshToken) IsExpired() bool {
	return time.Now().UTC().After(t.ExpiresAt)
}

// IsValid reports whether the token can still be used.
func (t *RefreshToken) IsValid() bool {
	return !t.Revoked && !t.IsExpired()
}
