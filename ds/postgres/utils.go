package postgres

import (
	"fmt"
	"strings"

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
