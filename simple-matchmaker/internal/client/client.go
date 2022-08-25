package mpclient

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/caarlos0/env"
)

const (
	authService = "cf"
	authRegion  = "eu-west-1"
)

// MultiplayClient represents something capable of interfacing with the multiplay API
type MultiplayClient interface {
	Allocate(fleet, region string, profile int64, uuid string) (*AllocateResponse, error)
	Allocations(fleet string, uuids ...string) ([]AllocationResponse, error)
	Deallocate(fleet, uuid string) error
}

// Config holds configuration used to access the multiplay api
type Config struct {
	AccessKey string `env:"MP_ACCESS_KEY"`
	SecretKey string `env:"MP_SECRET_KEY"`
	BaseURL   string `env:"MP_BASE_URL"`
}

// multiplayClient is the implementation of the multiplay client
type multiplayClient struct {
	accessKey string
	secretKey string
	baseURL   string
}

// NewClientFromEnv creates a multiplay client from the environment
func NewClientFromEnv() (MultiplayClient, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load multiplay config from env: %w", err)
	}

	if cfg.AccessKey == "" {
		return nil, fmt.Errorf("access key is empty")
	}

	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("access key is empty")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.multiplay.co.uk"
	}

	return NewClient(cfg), nil
}

// NewClient creates a multiplay client
func NewClient(cfg Config) MultiplayClient {
	return &multiplayClient{
		accessKey: cfg.AccessKey,
		secretKey: cfg.SecretKey,
		baseURL:   cfg.BaseURL,
	}
}

func getBody(w io.Reader) string {
	v, err := ioutil.ReadAll(w)
	if err != nil {
		return ""
	}
	return string(v)
}
