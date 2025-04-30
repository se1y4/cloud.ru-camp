package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Backends    []string `yaml:"backends"`
	RateLimiter struct {
		DefaultCapacity int           `yaml:"default_capacity"`
		DefaultRate     int           `yaml:"default_rate"`
		RefillInterval  time.Duration `yaml:"refill_interval"`
	} `yaml:"rate_limiter"`
	Balancer struct {
		Strategy            string        `yaml:"strategy"`
		HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	} `yaml:"balancer"`
	Postgres struct {
        ConnString string `yaml:"conn_string"`
    } `yaml:"postgres"`
}

func LoadConfig(path string) (*Config, error) {
	config := &Config{}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err := d.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}
