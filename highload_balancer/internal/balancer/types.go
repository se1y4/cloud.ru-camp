package balancer

import "time"

type StrategyType string

const (
	RoundRobinStrategy       StrategyType = "round-robin"
	LeastConnectionsStrategy StrategyType = "least-connections"
)

type Config struct {
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
}

type HealthCheckResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
