package config

// Config holds all gateway configuration.
type Config struct {
	Name           string         `yaml:"name"`
	Version        string         `yaml:"version"`
	Gateway        GatewayConfig  `yaml:"gateway"`
	Infrastructure InfraConfig    `yaml:"infrastructure"`
	Agents         []AgentRef     `yaml:"agents"`
}

// GatewayConfig holds HTTP server settings.
type GatewayConfig struct {
	Port int        `yaml:"port"`
	CORS CORSConfig `yaml:"cors"`
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

// InfraConfig holds infrastructure connections.
type InfraConfig struct {
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
}

// RabbitMQConfig holds RabbitMQ connection settings.
type RabbitMQConfig struct {
	URL      string `yaml:"url"`
	Exchange string `yaml:"exchange"`
}

// AgentRef references an agent manifest to load.
type AgentRef struct {
	Name         string `yaml:"name"`
	ManifestPath string `yaml:"manifest_path"`
}
