package ratelimiter

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type ClientManager struct {
	storage     ClientStorage
	rateLimiter *RateLimiter
	clients     map[string]*ClientConfig
	mux         sync.RWMutex
}

func (cm *ClientManager) GetAllClients() map[string]*ClientConfig {
	cm.mux.RLock()
	defer cm.mux.RUnlock()

	clients := make(map[string]*ClientConfig)
	for id, config := range cm.clients {
		clients[id] = &ClientConfig{
			ClientID:    config.ClientID,
			Capacity:    config.Capacity,
			RatePerSec:  config.RatePerSec,
			CreatedAt:   config.CreatedAt,
			LastUpdated: config.LastUpdated,
		}
	}
	return clients
}

func NewClientManager(storage ClientStorage) *ClientManager {
	cm := &ClientManager{
		storage: storage,
		clients: make(map[string]*ClientConfig),
	}
	cm.loadInitialClients()
	return cm
}

func (cm *ClientManager) loadInitialClients() {
	clients, err := cm.storage.GetAllClients()
	if err != nil {
		log.Printf("Failed to load initial clients: %v", err)
		return
	}

	cm.mux.Lock()
	defer cm.mux.Unlock()
	cm.clients = clients
}

func (cm *ClientManager) AddClient(clientID string, capacity, rate int) error {
	client := &ClientConfig{
		ClientID:    clientID,
		Capacity:    capacity,
		RatePerSec:  rate,
		CreatedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	if err := cm.storage.SaveClient(client); err != nil {
		return err
	}

	cm.mux.Lock()
	defer cm.mux.Unlock()
	cm.clients[clientID] = client
	return nil
}

func (cm *ClientManager) RemoveClient(clientID string) error {
	cm.mux.RLock()
	_, exists := cm.clients[clientID]
	cm.mux.RUnlock()

	if !exists {
		return fmt.Errorf("client not found")
	}

	if err := cm.storage.DeleteClient(clientID); err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	cm.mux.Lock()
	delete(cm.clients, clientID)
	cm.mux.Unlock()

	if cm.rateLimiter != nil {
		cm.rateLimiter.RemoveBucket(clientID)
	}

	return nil
}

func (cm *ClientManager) GetClientConfig(clientID string) (*ClientConfig, bool) {
	cm.mux.RLock()
	defer cm.mux.RUnlock()
	config, exists := cm.clients[clientID]
	return config, exists
}

func (cm *ClientManager) HandleClientAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var config ClientConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := cm.AddClient(config.ClientID, config.Capacity, config.RatePerSec); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(config)

	case http.MethodDelete:
		clientID := r.URL.Query().Get("client_id")
		if clientID == "" {
			http.Error(w, "client_id parameter is required", http.StatusBadRequest)
			return
		}

		if err := cm.RemoveClient(clientID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rl *RateLimiter) AllowWithConfig(clientID string, config *ClientConfig) bool {
	rl.mux.RLock()
	bucket, exists := rl.buckets[clientID]
	rl.mux.RUnlock()

	if !exists {
		rl.mux.Lock()
		bucket = &TokenBucket{
			capacity:   config.Capacity,
			tokens:     config.Capacity,
			rate:       config.RatePerSec,
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
