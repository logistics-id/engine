package postgres

import (
	"fmt"

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
