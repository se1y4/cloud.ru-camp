package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/se1y4/highload-balancer/internal/balancer"
	"github.com/se1y4/highload-balancer/internal/config"
	"github.com/se1y4/highload-balancer/internal/ratelimiter"
	"github.com/se1y4/highload-balancer/internal/server"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	strategy := balancer.NewStrategy(balancer.StrategyType(cfg.Balancer.Strategy))
	lb := balancer.NewLoadBalancer(
		cfg.Backends,
		strategy,
		&balancer.Config{
			HealthCheckInterval: cfg.Balancer.HealthCheckInterval,
		},
	)

	rl := ratelimiter.NewRateLimiter(
		cfg.RateLimiter.DefaultCapacity,
		cfg.RateLimiter.DefaultRate,
		cfg.RateLimiter.RefillInterval,
	)
	defer rl.Stop()

	srv := server.NewServer(lb, rl)
	httpServer := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: srv,
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting server on port %s", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-shutdownChan
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}
