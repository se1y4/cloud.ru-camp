package server

import (
	"encoding/json"
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

func NewServer(balancer *balancer.LoadBalancer, rateLimiter *ratelimiter.RateLimiter, clientManager *ratelimiter.ClientManager) *Server {
	return &Server{
		balancer:      balancer,
		rateLimiter:   rateLimiter,
		clientManager: clientManager,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		log.Printf("%s %s processed in %v", r.Method, r.URL.Path, time.Since(start))
	}()

	switch {
	case utils.IsHealthCheckRequest(r):
		w.WriteHeader(http.StatusOK)
		return

	case r.URL.Path == "/api/clients":
		s.handleClientsAPI(w, r)
		return

	default:
		s.handleProxyRequest(w, r)
	}
}

func (s *Server) handleClientsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getClients(w, r)
	case http.MethodPost:
		s.createClient(w, r)
	case http.MethodDelete:
		s.deleteClient(w, r)
	case http.MethodPatch:
		s.patchClient(w, r)
	default:
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) getClients(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")

	if clientID != "" {
		client, exists := s.clientManager.GetClientConfig(clientID)
		if !exists {
			utils.WriteErrorResponse(w, http.StatusNotFound, "Client not found")
			return
		}
		utils.WriteJSONResponse(w, http.StatusOK, client)
		return
	}

	clients := s.clientManager.GetAllClients()
	utils.WriteJSONResponse(w, http.StatusOK, clients)
}

func (s *Server) createClient(w http.ResponseWriter, r *http.Request) {
	var config ratelimiter.ClientConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if config.ClientID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "client_id is required")
		return
	}

	if err := s.clientManager.AddClient(config.ClientID, config.Capacity, config.RatePerSec); err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to create client")
		return
	}

	client, _ := s.clientManager.GetClientConfig(config.ClientID)
	w.Header().Set("Location", "/api/clients?client_id="+config.ClientID)
	utils.WriteJSONResponse(w, http.StatusCreated, client)
}

func (s *Server) deleteClient(w http.ResponseWriter, r *http.Request) {
	var clientID string

	clientID = r.URL.Query().Get("client_id")

	if clientID == "" {
		var requestBody struct {
			ClientID string `json:"client_id"`
		}

		if err := json.NewDecoder(r.Body).Decode(&requestBody); err == nil {
			clientID = requestBody.ClientID
		}
	}

	if clientID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "client_id is required either in query params or JSON body")
		return
	}

	if err := s.clientManager.RemoveClient(clientID); err != nil {
		if err.Error() == "client not found" {
			utils.WriteErrorResponse(w, http.StatusNotFound, err.Error())
		} else {
			log.Printf("Error deleting client: %v", err)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to delete client")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleProxyRequest(w http.ResponseWriter, r *http.Request) {
	clientIP := utils.GetClientIP(r)
	clientConfig, exists := s.clientManager.GetClientConfig(clientIP)

	var allowed bool
	if exists {
		allowed = s.rateLimiter.AllowWithConfig(clientIP, clientConfig)
	} else {
		allowed = s.rateLimiter.Allow(clientIP)
	}

	if !allowed {
		utils.WriteJSONResponse(w, http.StatusTooManyRequests, ratelimiter.RateLimitResponse{
			Code:       http.StatusTooManyRequests,
			Message:    "Rate limit exceeded",
			RetryAfter: time.Second,
		})
		return
	}

	s.balancer.ServeHTTP(w, r)
}

func (s *Server) patchClient(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "client_id parameter is required")
		return
	}

	currentClient, exists := s.clientManager.GetClientConfig(clientID)
	if !exists {
		utils.WriteErrorResponse(w, http.StatusNotFound, "Client not found")
		return
	}

	var patchData struct {
		Capacity   *int `json:"capacity,omitempty"`
		RatePerSec *int `json:"rate_per_sec,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&patchData); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid patch data")
		return
	}

	if patchData.Capacity != nil {
		currentClient.Capacity = *patchData.Capacity
	}
	if patchData.RatePerSec != nil {
		currentClient.RatePerSec = *patchData.RatePerSec
	}

	if err := s.clientManager.UpdateClient(currentClient); err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to update client")
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, currentClient)
}
