package pagination

import (
	"fmt"
	"strings"
)

// OrderDirection represents the sorting order direction (ASC or DESC).
type OrderDirection string

const (
	// OrderAsc sorts records in ascending order.
	OrderAsc OrderDirection = "ASC"

	// OrderDesc sorts records in descending order.
	OrderDesc OrderDirection = "DESC"
)

// SortCriteria represents a single field sorting instruction.
type SortCriteria struct {
	// Field is the name of the attribute requested by the client.
	Field string `json:"field"`

	// Order is the sort direction (ASC or DESC).
	Order OrderDirection `json:"order"`
}

// Sort holds a list of ordered sorting instructions.
type Sort struct {
	// Criteria contains the sequence of sort instructions applied to the query.
	Criteria []SortCriteria `json:"criteria"`
}

// ParseSort parses a sorting query string into a structured Sort definition.
//
// Supported formats:
// 1. Delimited pairs: "created_at:desc,id:asc" or "created_at.desc,id.asc"
// 2. Prefixed notation: "-created_at,+id,name" (minus means DESC, plus or bare means ASC)
// 3. SQL-style: "created_at desc, id asc"
//
// If the parsed string yields no valid criteria, defaultField and defaultOrder are applied.
//
// Example usage:
//
//	sort := pagination.ParseSort(r.URL.Query().Get("sort"), "id", pagination.OrderDesc)
func ParseSort(query string, defaultField string, defaultOrder OrderDirection) Sort {
	query = strings.TrimSpace(query)
	var criteria []SortCriteria

	if query != "" {
		tokens := strings.Split(query, ",")
		for _, token := range tokens {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}

			// Format 1: -field or +field
			if strings.HasPrefix(token, "-") {
				field := strings.TrimPrefix(token, "-")
				if field != "" {
					criteria = append(criteria, SortCriteria{Field: field, Order: OrderDesc})
					continue
				}
			} else if strings.HasPrefix(token, "+") {
				field := strings.TrimPrefix(token, "+")
				if field != "" {
					criteria = append(criteria, SortCriteria{Field: field, Order: OrderAsc})
					continue
				}
			}

			// Format 2: field:desc or field.desc or field_desc
			var field string
			order := OrderAsc

			if idx := strings.IndexAny(token, ":. "); idx != -1 {
				field = strings.TrimSpace(token[:idx])
				dirStr := strings.ToUpper(strings.TrimSpace(token[idx+1:]))
				if dirStr == "DESC" {
					order = OrderDesc
				}
			} else {
				field = token
			}

			if field != "" {
				criteria = append(criteria, SortCriteria{Field: field, Order: order})
			}
		}
	}

	// Fallback to default if no criteria parsed
	if len(criteria) == 0 && defaultField != "" {
		if defaultOrder != OrderDesc {
			defaultOrder = OrderAsc
		}
		criteria = append(criteria, SortCriteria{Field: defaultField, Order: defaultOrder})
	}

	return Sort{Criteria: criteria}
}

// SQL converts the Sort criteria into a sanitized, injection-safe SQL ORDER BY clause.
//
// Security Guarantee:
// Every field requested by the client is strictly checked against the provided whitelist map.
// If a field is not present in the whitelist, it is completely omitted.
//
// Example usage:
//
//	allowed := map[string]string{
//	    "id":         "users.id",
//	    "name":       "users.name",
//	    "created_at": "users.created_at",
//	}
//	orderClause := sort.SQL(allowed) // e.g. "users.created_at DESC, users.id ASC"
func (s Sort) SQL(whitelist map[string]string) string {
	if len(whitelist) == 0 {
		return ""
	}

	var clauses []string
	seen := make(map[string]bool)

	for _, c := range s.Criteria {
		// Look up safe DB column name from whitelist
		dbCol, ok := whitelist[strings.ToLower(c.Field)]
		if !ok || seen[dbCol] {
			continue // Skip unapproved or duplicate columns
		}

		seen[dbCol] = true
		dir := "ASC"
		if c.Order == OrderDesc {
			dir = "DESC"
		}

		clauses = append(clauses, fmt.Sprintf("%s %s", dbCol, dir))
	}

	return strings.Join(clauses, ", ")
}
