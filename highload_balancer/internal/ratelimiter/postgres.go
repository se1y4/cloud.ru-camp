package ratelimiter

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type ClientStorage interface {
	SaveClient(*ClientConfig) error
	DeleteClient(string) error
	GetClient(string) (*ClientConfig, error)
	GetAllClients() (map[string]*ClientConfig, error)
	Close() error
}

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(connStr string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) InitSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS clients (
		client_id TEXT PRIMARY KEY,
		capacity INTEGER NOT NULL,
		rate_per_sec INTEGER NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);
	
	CREATE INDEX IF NOT EXISTS idx_clients_updated ON clients(updated_at);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStorage) SaveClient(client *ClientConfig) error {
	query := `
	INSERT INTO clients (client_id, capacity, rate_per_sec)
	VALUES ($1, $2, $3)
	ON CONFLICT (client_id) 
	DO UPDATE SET 
		capacity = EXCLUDED.capacity,
		rate_per_sec = EXCLUDED.rate_per_sec,
		updated_at = NOW()
	`
	_, err := s.db.Exec(query, 
		client.ClientID, 
		client.Capacity, 
		client.RatePerSec,
	)
	return err
}

func (s *PostgresStorage) DeleteClient(clientID string) error {
	_, err := s.db.Exec(
		"DELETE FROM clients WHERE client_id = $1", 
		clientID,
	)
	return err
}

func (s *PostgresStorage) GetClient(clientID string) (*ClientConfig, error) {
	var config ClientConfig
	
	err := s.db.QueryRow(`
		SELECT 
			client_id, 
			capacity, 
			rate_per_sec, 
			created_at, 
			updated_at 
		FROM clients 
		WHERE client_id = $1
	`, clientID).Scan(
		&config.ClientID,
		&config.Capacity,
		&config.RatePerSec,
		&config.CreatedAt,
		&config.LastUpdated,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

func (s *PostgresStorage) GetAllClients() (map[string]*ClientConfig, error) {
	rows, err := s.db.Query(`
		SELECT 
			client_id, 
			capacity, 
			rate_per_sec, 
			created_at, 
			updated_at 
		FROM clients
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clients := make(map[string]*ClientConfig)
	for rows.Next() {
		var config ClientConfig
		if err := rows.Scan(
			&config.ClientID,
			&config.Capacity,
			&config.RatePerSec,
			&config.CreatedAt,
			&config.LastUpdated,
		); err != nil {
			return nil, err
		}
		clients[config.ClientID] = &config
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return clients, nil
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}