package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jhaveripatric/agent-gateway/internal/auth"
	"github.com/jhaveripatric/agent-gateway/internal/config"
	"github.com/jhaveripatric/agent-gateway/internal/manifest"
	"github.com/jhaveripatric/agent-gateway/internal/middleware"
	"github.com/jhaveripatric/agent-gateway/internal/router"
	"github.com/jhaveripatric/agent-gateway/internal/rpc"
)

// Server is the HTTP gateway server.
type Server struct {
	cfg         *config.Config
	router      chi.Router
	rpcClient   *rpc.Client
	jwtVerifier *auth.JWTVerifier
	manifests   []manifest.Manifest
}

// New creates a new gateway server.
func New(cfg *config.Config) (*Server, error) {
	s := &Server{cfg: cfg}

	// Initialize RPC client
	rpcClient, err := rpc.NewClient(rpc.Config{
		URL:      cfg.Infrastructure.RabbitMQ.URL,
		Exchange: cfg.Infrastructure.RabbitMQ.Exchange,
	})
	if err != nil {
		return nil, fmt.Errorf("init rpc client: %w", err)
	}
	s.rpcClient = rpcClient
	log.Printf("Connected to RabbitMQ at %s", cfg.Infrastructure.RabbitMQ.URL)

	// Initialize JWT verifier
	s.jwtVerifier = auth.NewJWTVerifier("agenteco", "agent-gateway")

	if err := s.loadManifests(); err != nil {
		return nil, err
	}

	// Load public keys from manifests
	if err := s.loadJWTKeys(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) loadManifests() error {
	loader := manifest.NewLoader(".")

	for _, agent := range s.cfg.Agents {
		m, err := loader.Load(agent.ManifestPath)
		if err != nil {
			log.Printf("Warning: failed to load %s manifest: %v", agent.Name, err)
			continue
		}
		log.Printf("Loaded manifest: %s v%s (%d actions)",
			m.Name, m.Version, len(m.Actions))
		s.manifests = append(s.manifests, *m)
	}

	s.router = s.buildRouter(s.manifests)
	return nil
}

func (s *Server) loadJWTKeys() error {
	for _, m := range s.manifests {
		if m.JWT != nil && m.JWT.PublicKeyPath != "" {
			// Resolve path relative to manifest location
			keyPath := m.JWT.PublicKeyPath
			if !filepath.IsAbs(keyPath) {
				keyPath = filepath.Join(filepath.Dir(m.ManifestPath), keyPath)
			}

			keyID := m.JWT.KeyID
			if keyID == "" {
				keyID = m.Name + "-v1"
			}

			if err := s.jwtVerifier.LoadPublicKey(keyID, keyPath); err != nil {
				log.Printf("Warning: failed to load public key for %s: %v", m.Name, err)
				continue
			}
			log.Printf("Loaded public key: %s from %s", keyID, keyPath)
		}
	}

	if !s.jwtVerifier.HasKeys() {
		log.Printf("Warning: no JWT public keys loaded - auth:bearer routes will fail")
	}

	return nil
}

func (s *Server) buildRouter(manifests []manifest.Manifest) chi.Router {
	r := chi.NewRouter()

	// Middleware stack (order matters)
	r.Use(middleware.RequestID)
	r.Use(middleware.Security)
	r.Use(middleware.Recovery)
	r.Use(cors.Handler(middleware.CORSOptions(s.cfg.Gateway.CORS.AllowedOrigins)))
	r.Use(middleware.Logger)

	// Health endpoints
	r.Get("/healthz", s.healthHandler)
	r.Get("/readyz", s.readyHandler)

	// Mount agent routes with RPC and JWT
	builder := router.NewBuilder(s.rpcClient, s.jwtVerifier)
	agentRoutes := builder.Build(manifests)
	r.Mount("/", agentRoutes)

	return r
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !s.rpcClient.Ready() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not_ready",
			"reason": "rabbitmq disconnected",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.cfg.Gateway.Port)
	log.Printf("Starting agent-gateway on %s", addr)
	return http.ListenAndServe(addr, s.router)
}

// Close shuts down the server and connections.
func (s *Server) Close() error {
	if s.rpcClient != nil {
		return s.rpcClient.Close()
	}
	return nil
}
