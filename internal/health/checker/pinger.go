package checker

import (
	"context"

	"gorm.io/gorm"
)

// Pinger defines the contract for verifying dependency readiness.
type Pinger interface {
	Ping(ctx context.Context) error
}

// DBPinger verifies database connectivity.
type DBPinger struct {
	db *gorm.DB
}

// NewDBPinger creates a new DBPinger.
func NewDBPinger(db *gorm.DB) *DBPinger {
	return &DBPinger{db: db}
}

// Ping checks if the database connection pool is healthy.
func (p *DBPinger) Ping(ctx context.Context) error {
	if p.db == nil {
		return nil
	}
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
