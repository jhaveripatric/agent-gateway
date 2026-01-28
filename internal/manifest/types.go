package manifest

import "time"

// Manifest represents an agent's capabilities and routes.
type Manifest struct {
	Name        string     `yaml:"name"`
	Version     string     `yaml:"version"`
	Description string     `yaml:"description"`
	JWT         *JWTConfig `yaml:"jwt,omitempty"`
	Actions     []Action   `yaml:"actions"`
}

// JWTConfig holds JWT validation settings.
type JWTConfig struct {
	Algorithm string `yaml:"algorithm"`
	PublicKey string `yaml:"public_key"`
	Issuer    string `yaml:"issuer"`
	Audience  string `yaml:"audience"`
}

// Action represents a single API action.
type Action struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	HTTP        HTTPConfig     `yaml:"http"`
	Auth        string         `yaml:"auth"`
	Permission  string         `yaml:"permission"`
	RateLimit   string         `yaml:"rate_limit"`
	Timeout     time.Duration  `yaml:"timeout"`
	Request     RequestConfig  `yaml:"request"`
	Response    ResponseConfig `yaml:"response"`
}

// HTTPConfig defines HTTP method and path.
type HTTPConfig struct {
	Method string `yaml:"method"`
	Path   string `yaml:"path"`
}

// RequestConfig defines the request event and schema.
type RequestConfig struct {
	Event  string         `yaml:"event"`
	Schema map[string]any `yaml:"schema"`
}

// ResponseConfig defines response mappings.
type ResponseConfig struct {
	Success ResponseMapping `yaml:"success"`
	Failure ResponseMapping `yaml:"failure"`
	Timeout ResponseMapping `yaml:"timeout"`
}

// ResponseMapping maps events to HTTP responses.
type ResponseMapping struct {
	Event  string         `yaml:"event"`
	Status int            `yaml:"status"`
	Body   map[string]any `yaml:"body,omitempty"`
}
