package ratelimiter

import (
	"sync"
	"time"
)

type TokenBucket struct {
	capacity   int
	tokens     int
	rate       int 
	lastRefill time.Time
	mux        sync.Mutex
}

type RateLimiter struct {
	buckets     map[string]*TokenBucket
	defaultCap  int
	defaultRate int
	mux         sync.RWMutex
	stopChan    chan struct{}
}

func NewRateLimiter(defaultCap, defaultRate int, refillInterval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets:     make(map[string]*TokenBucket),
		defaultCap:  defaultCap,
		defaultRate: defaultRate,
		stopChan:    make(chan struct{}),
	}

	go rl.autoRefill(refillInterval)
	return rl
}

func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}

func (rl *RateLimiter) autoRefill(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.refillAllBuckets()
		case <-rl.stopChan:
			return
		}
	}
}

func (rl *RateLimiter) refillAllBuckets() {
	rl.mux.Lock()
	defer rl.mux.Unlock()

	for _, bucket := range rl.buckets {
		bucket.refill()
	}
}

func (tb *TokenBucket) refill() {
	tb.mux.Lock()
	defer tb.mux.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokensToAdd := int(elapsed * float64(tb.rate))

	if tokensToAdd > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}
}

func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mux.RLock()
	bucket, exists := rl.buckets[clientID]
	rl.mux.RUnlock()

	if !exists {
		rl.mux.Lock()
		bucket = &TokenBucket{
			capacity:   rl.defaultCap,
			tokens:     rl.defaultCap,
			rate:       rl.defaultRate,
			lastRefill: time.Now(),
		}
		rl.buckets[clientID] = bucket
		rl.mux.Unlock()
	}

	bucket.mux.Lock()
	defer bucket.mux.Unlock()

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
