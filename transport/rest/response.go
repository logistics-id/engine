package rest

// Response defines the standard structure for all HTTP responses
type ResponseBody struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Errors  any    `json:"errors,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

type Meta struct {
	Page       int  `json:"page"`
	PageSize   int  `json:"page_size"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

func BuildMeta(page, pageSize, total int) *Meta {
	totalPages := (total + pageSize - 1) / pageSize // ceil division
	hasNext := page < totalPages
	hasPrev := page > 1

	return &Meta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}
}

func NewResponseBody(metas ...*Meta) *ResponseBody {
	rb := &ResponseBody{}

	if len(metas) > 0 {
		rb.Meta = metas[0]
	}

	return rb
}
