package rest

import (
	"bytes"
	"net"
	"net/http"
	"strings"

	"github.com/jinzhu/copier"
)

type GetRequest struct {
	Limit   int      `query:"limit"`
	Page    int      `query:"page"`
	Search  string   `query:"search"`
	OrderBy string   `query:"order_by"`
	Offset  int      `query:"-"`
	Orders  []string `query:"-"`
}

func (r *GetRequest) GetLimit() int {
	if r.Limit == 0 {
		r.Limit = 25
	}

	return r.Limit
}

func (r *GetRequest) GetSearch() string {
	return r.Search
}

func (r *GetRequest) GetOffset() int {
	if r.Page == 0 {
		r.Page = 1
	}

	return r.GetLimit() * (r.Page - 1)
}

func (r *GetRequest) GetPage() int {
	if r.Page == 0 {
		r.Page = 1
	}

	return r.Page
}

func (r *GetRequest) GetOrders() []string {
	if r.OrderBy == "" {
		return []string{"-id"}
	}

	return strings.Split(strings.ReplaceAll(r.OrderBy, ".", "__"), ",")
}

func (r *GetRequest) Copy(x any) {
	r.Limit = r.GetLimit()
	r.Offset = r.GetOffset()
	r.Orders = r.GetOrders()

	copier.Copy(x, r)
}

func (r *GetRequest) GetMetaResponse(total int) *Meta {
	return BuildMeta(r.GetPage(), r.GetLimit(), total)
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rw *responseRecorder) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseRecorder) Write(b []byte) (int, error) {
	rw.body.Write(b) // capture for log
	return rw.ResponseWriter.Write(b)
}

func getRealIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// This can be a comma-separated list of IPs
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // return full if can't split
	}
	return ip
}
