package server

import (
	"log"
	"net/http"
	"time"

	"github.com/se1y4/highload-balancer/internal/balancer"
	"github.com/se1y4/highload-balancer/internal/ratelimiter"
	"github.com/se1y4/highload-balancer/utils"
)

type Server struct {
	balancer      *balancer.LoadBalancer
	rateLimiter   *ratelimiter.RateLimiter
	clientManager *ratelimiter.ClientManager
}

func NewServer(balancer *balancer.LoadBalancer, rateLimiter *ratelimiter.RateLimiter) *Server {
	return &Server{
		balancer:      balancer,
		rateLimiter:   rateLimiter,
		clientManager: ratelimiter.NewClientManager(),
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if utils.IsHealthCheckRequest(r) {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.URL.Path == "/clients" {
		s.clientManager.HandleClientAPI(w, r)
		return
	}

	clientIP := utils.GetClientIP(r)
	if !s.rateLimiter.Allow(clientIP) {
		utils.WriteJSONResponse(w, http.StatusTooManyRequests, ratelimiter.RateLimitResponse{
			Code:       http.StatusTooManyRequests,
			Message:    "Rate limit exceeded",
			RetryAfter: time.Second,
		})
		return
	}

	start := time.Now()
	s.balancer.ServeHTTP(w, r)
	log.Printf("Request processed in %v", time.Since(start))
}
