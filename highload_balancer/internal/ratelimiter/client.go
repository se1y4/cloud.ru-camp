package ratelimiter

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type ClientManager struct {
	clients map[string]*ClientConfig
	mux     sync.RWMutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]*ClientConfig),
	}
}

func (cm *ClientManager) AddClient(clientID string, capacity, rate int) {
	cm.mux.Lock()
	defer cm.mux.Unlock()

	now := time.Now()
	if client, exists := cm.clients[clientID]; exists {
		client.Capacity = capacity
		client.RatePerSec = rate
		client.LastUpdated = now
	} else {
		cm.clients[clientID] = &ClientConfig{
			ClientID:    clientID,
			Capacity:    capacity,
			RatePerSec:  rate,
			CreatedAt:   now,
			LastUpdated: now,
		}
	}
}

func (cm *ClientManager) RemoveClient(clientID string) {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	delete(cm.clients, clientID)
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

		cm.AddClient(config.ClientID, config.Capacity, config.RatePerSec)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(config)

	case http.MethodDelete:
		clientID := r.URL.Query().Get("client_id")
		if clientID == "" {
			http.Error(w, "client_id parameter is required", http.StatusBadRequest)
			return
		}

		cm.RemoveClient(clientID)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
