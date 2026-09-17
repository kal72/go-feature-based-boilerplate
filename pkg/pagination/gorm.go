package pagination

import (
	"gorm.io/gorm"
)

// GormScope returns a GORM scope function that applies Offset and Limit based on Params.
//
// Example usage:
//
//	var users []entity.User
//	err := db.Scopes(pagination.GormScope(params)).Find(&users).Error
func GormScope(p Params) func(db *gorm.DB) *gorm.DB {
	p.Normalize()
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(p.Offset()).Limit(p.Limit())
	}
}

// GormSortScope returns a GORM scope function that applies a sanitized ORDER BY clause.
// If the sort criteria produce no valid whitelisted columns, no ORDER clause is appended.
//
// Example usage:
//
//	whitelist := map[string]string{
//	    "id":         "users.id",
//	    "name":       "users.name",
//	    "created_at": "users.created_at",
//	}
//	err := db.Scopes(
//	    pagination.GormScope(params),
//	    pagination.GormSortScope(sort, whitelist),
//	).Find(&users).Error
func GormSortScope(s Sort, whitelist map[string]string) func(db *gorm.DB) *gorm.DB {
	orderClause := s.SQL(whitelist)
	return func(db *gorm.DB) *gorm.DB {
		if orderClause != "" {
			return db.Order(orderClause)
		}
		return db
	}
}
