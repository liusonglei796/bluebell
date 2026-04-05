package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// httpRequestsTotal tracks total HTTP requests by method, path, and status code.
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// httpRequestDurationSeconds tracks HTTP request duration in seconds by method and path.
	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

// RegisterPrometheusMetrics registers all Prometheus HTTP metrics collectors.
// Safe to call multiple times — uses Register (not MustRegister) to avoid duplicates.
func RegisterPrometheusMetrics() {
	prometheus.Register(httpRequestsTotal)
	prometheus.Register(httpRequestDurationSeconds)
}

// PrometheusMiddleware returns a gin.HandlerFunc that records HTTP request metrics.
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Execute the request chain
		c.Next()

		// Record metrics after the request completes
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.FullPath() // Use matched route pattern (e.g., /api/v1/post/:id) instead of raw URL

		// Fallback to raw path if FullPath is empty (e.g., 404 routes)
		if path == "" {
			path = c.Request.URL.Path
		}

		httpRequestsTotal.WithLabelValues(method, path, statusToCode(status)).Inc()
		httpRequestDurationSeconds.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
	}
}

// 它被注册为一个 GET 路由，路径是 /metrics。当 Prometheus 服务定期访问 http://你的服务:8080/metrics 时，就会触发这个函数。
// PrometheusHandler returns an http.Handler that serves the /metrics endpoint.
func PrometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// statusToCode converts HTTP status code to string for label values.
func statusToCode(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	case status >= 200:
		return "2xx"
	default:
		return "1xx"
	}
}
