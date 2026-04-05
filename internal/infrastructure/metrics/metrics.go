package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds custom business metrics for the bluebell application.
type Metrics struct {
	ActiveUsers  prometheus.Gauge
	TotalVotes   prometheus.Counter
	TotalPosts   prometheus.Counter
	AuditResults *prometheus.CounterVec
}

// RegisterCustomMetrics creates and registers all custom business metrics.
// Returns a Metrics struct with convenient methods for updating metrics.
// Safe to call multiple times — uses Register (not MustRegister) to avoid duplicates.
func RegisterCustomMetrics() *Metrics {
	activeUsers := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "active_users",
		Help: "Current number of active users",
	})

	totalVotes := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_votes",
		Help: "Total number of votes cast",
	})

	totalPosts := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_posts",
		Help: "Total number of posts created",
	})

	auditResults := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "audit_results",
			Help: "Audit result statistics",
		},
		[]string{"result"},
	)

	// Register all metrics (Register returns false if already registered, avoiding panic)
	prometheus.Register(activeUsers)
	prometheus.Register(totalVotes)
	prometheus.Register(totalPosts)
	prometheus.Register(auditResults)

	return &Metrics{
		ActiveUsers:  activeUsers,
		TotalVotes:   totalVotes,
		TotalPosts:   totalPosts,
		AuditResults: auditResults,
	}
}

// SetActiveUsers sets the current number of active users.
func (m *Metrics) SetActiveUsers(count float64) {
	m.ActiveUsers.Set(count)
}

// IncrVotes increments the total vote counter by 1.
func (m *Metrics) IncrVotes() {
	m.TotalVotes.Inc()
}

// IncrPosts increments the total post counter by 1.
func (m *Metrics) IncrPosts() {
	m.TotalPosts.Inc()
}

// IncrAuditResult increments the audit result counter for the given result label.
func (m *Metrics) IncrAuditResult(result string) {
	m.AuditResults.WithLabelValues(result).Inc()
}
