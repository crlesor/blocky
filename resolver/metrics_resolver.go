package resolver

import (
	"blocky/config"
	"fmt"
)

// MetricsResolver resolver that records metrics about requests/response
type MetricsResolver struct {
	NextResolver
	cfg     config.PrometheusConfig
	metrics Metrics
}

func (m MetricsResolver) handleMetrics(req *Request, resp *Response) {
	if m.cfg.Enable {
		m.metrics.RecordStats(req, resp)
	}
}

// Resolve resolves the passed request
func (m MetricsResolver) Resolve(req *Request) (*Response, error) {
	resp, err := m.next.Resolve(req)

	m.handleMetrics(req, resp)

	return resp, err
}

// Configuration gets the config of this resolver in a string slice
func (m MetricsResolver) Configuration() (result []string) {
	result = append(result, "metrics:")
	result = append(result, fmt.Sprintf("  Enable = %t", m.cfg.Enable))
	result = append(result, fmt.Sprintf("  Port   = %d", m.cfg.Port))
	result = append(result, fmt.Sprintf("  Path   = %s", m.cfg.Path))

	return
}

func (m MetricsResolver) String() string {
	return "metrics resolver"
}

// NewMetricsResolver creates a new intance of the MetricsResolver type
func NewMetricsResolver(cfg config.PrometheusConfig) MetricsResolver {
	if cfg.Path == "" {
		cfg.Path = "/metrics"
	}

	if cfg.Port == 0 {
		cfg.Port = 4000
	}

	metrics := NewMetrics(cfg)
	metrics.Start()

	return MetricsResolver{cfg: cfg, metrics: metrics}
}
