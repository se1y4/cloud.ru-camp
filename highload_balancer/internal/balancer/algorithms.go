package balancer

import "sync/atomic"

type Strategy interface {
	GetNextBackend(backends []*Backend) *Backend
}

type RoundRobin struct {
	counter uint64
}

func (r *RoundRobin) GetNextBackend(backends []*Backend) *Backend {
	next := atomic.AddUint64(&r.counter, 1)
	return backends[next%uint64(len(backends))]
}

type LeastConnections struct {
}

func (l *LeastConnections) GetNextBackend(backends []*Backend) *Backend {
	var best *Backend
	for _, b := range backends {
		if !b.IsAlive() {
			continue
		}
		if best == nil {
			best = b
			continue
		}
	}
	return best
}