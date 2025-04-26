package balancer

import (
	"context"
	"net/http"
	"time"
)

type HealthChecker struct {
	interval time.Duration
	timeout  time.Duration
	stopChan chan struct{}
}

func NewHealthChecker(interval, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		interval: interval,
		timeout:  timeout,
		stopChan: make(chan struct{}),
	}
}

func (hc *HealthChecker) Start(backends []*Backend) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.checkAllBackends(backends)
		case <-hc.stopChan:
			return
		}
	}
}

func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
}

func (hc *HealthChecker) checkAllBackends(backends []*Backend) {
	for _, backend := range backends {
		go func(b *Backend) {
			alive := hc.checkBackend(b.URL.String())
			b.SetAlive(alive)
		}(backend)
	}
}

func (hc *HealthChecker) checkBackend(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
