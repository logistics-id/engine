package common

import "strings"

type QueryOption struct {
	Limit      int64    `query:"limit"`
	Page       int64    `query:"page"`
	Search     string   `query:"search"`
	OrderBy    string   `query:"order_by"`
	Offset     int64    `query:"-"`
	Orders     []string `query:"-"`
	Conditions []any    `query:"-"`
}

func (r *QueryOption) GetLimit() int64 {
	if r.Limit == 0 {
		r.Limit = 25
	}

	return r.Limit
}

func (r *QueryOption) GetSearch() string {
	return r.Search
}

func (r *QueryOption) GetOffset() int64 {
	if r.Page == 0 {
		r.Page = 1
	}

	return r.GetLimit() * (r.Page - 1)
}

func (r *QueryOption) GetPage() int64 {
	if r.Page == 0 {
		r.Page = 1
	}

	return r.Page
}

func (r *QueryOption) GetOrders() []string {
	if r.OrderBy == "" {
		return []string{"-id"}
	}

	return strings.Split(strings.ReplaceAll(r.OrderBy, ".", "__"), ",")
}

func (r *QueryOption) BuildOption() *QueryOption {
	return r
}
