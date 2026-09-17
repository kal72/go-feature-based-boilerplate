package entity

import "time"

// Role represents the user's access level.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// User is the core domain entity for the user feature.
// Following a pragmatic Lightweight DDD approach, GORM tags are embedded directly
// to avoid excessive mapping boilerplate between domain entities and persistence models.
type User struct {
	ID        uint      `gorm:"primarykey"`
	Name      string    `gorm:"not null"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Password  string    `gorm:"not null"`
	Role      Role      `gorm:"not null;default:user"`
	Active    bool      `gorm:"not null;default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName tells GORM which table to use for this entity.
func (User) TableName() string { return "users" }
