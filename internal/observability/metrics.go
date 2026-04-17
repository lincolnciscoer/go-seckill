package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_seckill_http_requests_total",
			Help: "Total number of HTTP requests handled by the API.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "go_seckill_http_request_duration_seconds",
			Help:    "Latency of HTTP requests handled by the API.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	seckillAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_seckill_seckill_attempts_total",
			Help: "Total number of seckill attempts by result.",
		},
		[]string{"result"},
	)

	mqConsumeTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_seckill_mq_consume_total",
			Help: "Total number of MQ consume attempts by result.",
		},
		[]string{"result"},
	)

	orderStatusTransitionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_seckill_order_status_transitions_total",
			Help: "Total number of order status transitions by status.",
		},
		[]string{"status"},
	)
)

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func HTTPMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		status := strconv.Itoa(c.Writer.Status())
		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path, status).Observe(time.Since(start).Seconds())
	}
}

func RecordSeckillAttempt(result string) {
	seckillAttemptsTotal.WithLabelValues(result).Inc()
}

func RecordMQConsume(result string) {
	mqConsumeTotal.WithLabelValues(result).Inc()
}

func RecordOrderStatus(status string) {
	orderStatusTransitionsTotal.WithLabelValues(status).Inc()
}
