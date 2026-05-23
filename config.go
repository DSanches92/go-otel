package gotel

import (
	"errors"
	"time"
)

// ---- Erros Públicos

var (
	ErrServiceNameRequired       = errors.New("gotel: ServiceName é obrigatório")
	ErrCollectorEndpointRequired = errors.New("gotel: CollectorEndpoint é obrigatório")
	ErrAtLeastOneSignalRequired  = errors.New("gotel: ao menos um sinal deve ser habilitado (WithTracing, WithMetrics ou WithLogging)")
)

// ---- Configuração Default

type Config struct {
	ServiceName       string
	CollectorEndpoint string

	ServiceVersion string
	Environment    string
	Timeout        time.Duration
	Insecure       bool

	TracingEnabled bool
	MetricsEnabled bool
	LoggingEnabled bool
}

type Option func(*Config)

func NewConfig(options ...Option) *Config {
	config := configWithDefaults()

	for _, option := range options {
		option(config)
	}

	return config
}

func (config *Config) Validate() error {
	if config.ServiceName == "" {
		return ErrServiceNameRequired
	}

	if config.CollectorEndpoint == "" {
		return ErrCollectorEndpointRequired
	}

	if !config.TracingEnabled && !config.MetricsEnabled && !config.LoggingEnabled {
		return ErrAtLeastOneSignalRequired
	}

	return nil
}

func configWithDefaults() *Config {
	return &Config{
		ServiceVersion: "0.0.0",
		Environment:    "development",
		Timeout:        5 * time.Second,
		Insecure:       false,
	}
}

// ---- Obrigatórios

func WithServiceName(name string) Option {
	return func(config *Config) {
		config.ServiceName = name
	}
}

func WithCollectorEndpoint(endpoint string) Option {
	return func(config *Config) {
		config.CollectorEndpoint = endpoint
	}
}

// ---- Opcionais

func WithServiceVersion(version string) Option {
	return func(config *Config) {
		config.ServiceVersion = version
	}
}

func WithEnvironment(env string) Option {
	return func(config *Config) {
		config.Environment = env
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(config *Config) {
		config.Timeout = timeout
	}
}

func WithInsecure(insecure bool) Option {
	return func(config *Config) {
		config.Insecure = insecure
	}
}

// ---- Sinais

func WithTracing() Option {
	return func(config *Config) {
		config.TracingEnabled = true
	}
}

func WithMetrics() Option {
	return func(config *Config) {
		config.MetricsEnabled = true
	}
}

func WithLogging() Option {
	return func(config *Config) {
		config.LoggingEnabled = true
	}
}
