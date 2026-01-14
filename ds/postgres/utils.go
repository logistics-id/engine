package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/uptrace/bun"
)

// FilterSearch adds a case-insensitive search filter for the given fields.
func FilterSearch(q *bun.SelectQuery, search string, fields ...string) {
	q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		for _, field := range fields {
			q = q.WhereOr(fmt.Sprintf("%s ILIKE ?", field), fmt.Sprintf("%%%s%%", search))
		}
		return q
	})
}

func RequestSort(sort []string) string {
	var result []string

	for _, s := range sort {
		order := "ASC"
		if strings.HasPrefix(s, "-") {
			order = "DESC"
			s = s[1:]
		}

		s = strings.ReplaceAll(s, ":", ".")
		parts := strings.Split(s, "__")

		field := parts[0]
		for _, p := range parts[1:] {
			field += fmt.Sprintf("->>'%s'", p)
		}

		result = append(result, fmt.Sprintf("%s %s", field, order))
	}

	return strings.Join(result, ", ")
}

// PostgreSQL error codes (SQLSTATE)
const (
	// Class 23 — Integrity Constraint Violation
	ErrCodeUniqueViolation     = "23505" // unique_violation
	ErrCodeForeignKeyViolation = "23503" // foreign_key_violation
	ErrCodeNotNullViolation    = "23502" // not_null_violation
	ErrCodeCheckViolation      = "23514" // check_violation
	ErrCodeExclusionViolation  = "23P01" // exclusion_violation
)

// getPQError extracts a pq.Error from an error, handling wrapped errors.
func getPQError(err error) (*pq.Error, bool) {
	if err == nil {
		return nil, false
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr, true
	}

	return nil, false
}

// IsUniqueViolation checks if error is a PostgreSQL unique constraint violation.
// Handles both direct pq.Error and wrapped errors (e.g., from Bun ORM).
func IsUniqueViolation(err error) bool {
	pqErr, ok := getPQError(err)
	if !ok {
		return false
	}
	return pqErr.Code == ErrCodeUniqueViolation
}

// IsForeignKeyViolation checks if error is a PostgreSQL foreign key constraint violation.
func IsForeignKeyViolation(err error) bool {
	pqErr, ok := getPQError(err)
	if !ok {
		return false
	}
	return pqErr.Code == ErrCodeForeignKeyViolation
}

// IsNotNullViolation checks if error is a PostgreSQL NOT NULL constraint violation.
func IsNotNullViolation(err error) bool {
	pqErr, ok := getPQError(err)
	if !ok {
		return false
	}
	return pqErr.Code == ErrCodeNotNullViolation
}

// IsCheckViolation checks if error is a PostgreSQL CHECK constraint violation.
func IsCheckViolation(err error) bool {
	pqErr, ok := getPQError(err)
	if !ok {
		return false
	}
	return pqErr.Code == ErrCodeCheckViolation
}

// GetPostgresErrorCode returns the PostgreSQL error code (SQLSTATE) if the error is a pq.Error.
// Returns empty string if not a PostgreSQL error.
func GetPostgresErrorCode(err error) string {
	pqErr, ok := getPQError(err)
	if !ok {
		return ""
	}
	return string(pqErr.Code)
}

// GetPostgresErrorConstraint returns the constraint name from a PostgreSQL error.
// Useful for identifying which constraint was violated.
func GetPostgresErrorConstraint(err error) string {
	pqErr, ok := getPQError(err)
	if !ok {
		return ""
	}
	return pqErr.Constraint
}
