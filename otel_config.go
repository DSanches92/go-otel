package gotel

import (
	"errors"
	"os"
	"time"

	"go.opentelemetry.io/otel/sdk/trace"
)

var (
	ErrServiceNameRequired       = errors.New("gotel: ServiceName is required")
	ErrCollectorEndpointRequired = errors.New("gotel: CollectorEndpoint is required")
	ErrAtLeastOneSignalRequired  = errors.New("gotel: at least one signal must be enabled (WithTracing, WithMetrics or WithLogging)")
)

// ---- Configuração

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

	Sampler trace.Sampler
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

func WithEnvConfig() Option {
	return func(config *Config) {
		if config.ServiceName == "" {
			if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
				config.ServiceName = v
			}
		}

		if config.CollectorEndpoint == "" {
			if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
				config.CollectorEndpoint = v
			}
		}

		if config.Environment == "development" {
			if v := os.Getenv("APP_ENV"); v != "" {
				config.Environment = v
			}
		}

		if v := os.Getenv("OTEL_INSECURE"); v == "true" {
			config.Insecure = true
		}
	}
}

func WithSampler(sampler trace.Sampler) Option {
	return func(config *Config) {
		config.Sampler = sampler
	}
}
