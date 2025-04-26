package balancer

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

func NewStrategy(strategyType StrategyType) Strategy {
	switch strategyType {
	case RoundRobinStrategy:
		return &RoundRobin{}
	case LeastConnectionsStrategy:
		return &LeastConnections{}
	default:
		log.Printf("Unknown strategy type %s, defaulting to round-robin", strategyType)
		return &RoundRobin{}
	}
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	alive := b.Alive
	b.mux.RUnlock()
	return alive
}

type LoadBalancer struct {
	backends []*Backend
	current  uint64
	strategy Strategy
	config   *Config
}

func NewLoadBalancer(serverUrls []string, strategy Strategy, config *Config) *LoadBalancer {
	var backends []*Backend
	for _, serverUrl := range serverUrls {
		parsedUrl, err := url.Parse(serverUrl)
		if err != nil {
			log.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(parsedUrl)
		backends = append(backends, &Backend{
			URL:          parsedUrl,
			Alive:        true,
			ReverseProxy: proxy,
		})
	}

	lb := &LoadBalancer{
		backends: backends,
		strategy: strategy,
		config:   config,
	}

	if config != nil && config.HealthCheckInterval > 0 {
		lb.HealthCheck()
	}

	return lb
}

func (lb *LoadBalancer) HealthCheck() {
	for _, b := range lb.backends {
		go func(backend *Backend) {
			for {
				alive := isBackendAlive(backend.URL.String())
				backend.SetAlive(alive)
				time.Sleep(lb.config.HealthCheckInterval)
			}
		}(b)
	}
}

func (lb *LoadBalancer) GetNextBackend() *Backend {
	next := lb.strategy.GetNextBackend(lb.backends)
	attempts := len(lb.backends)
	for i := 0; i < attempts; i++ {
		if next.IsAlive() {
			return next
		}
		next = lb.strategy.GetNextBackend(lb.backends)
	}
	return nil
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := lb.GetNextBackend()
	if backend != nil {
		log.Printf("Routing request to %s", backend.URL.String())
		backend.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "No available backends", http.StatusServiceUnavailable)
}

func isBackendAlive(url string) bool {
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(url + "/health")
	if err != nil {
		log.Printf("Backend %s is down: %v", url, err)
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
