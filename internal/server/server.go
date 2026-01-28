package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jhaveripatric/agent-gateway/internal/config"
	"github.com/jhaveripatric/agent-gateway/internal/manifest"
	"github.com/jhaveripatric/agent-gateway/internal/middleware"
	"github.com/jhaveripatric/agent-gateway/internal/router"
	"github.com/jhaveripatric/agent-gateway/internal/rpc"
)

// Server is the HTTP gateway server.
type Server struct {
	cfg       *config.Config
	router    chi.Router
	rpcClient *rpc.Client
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

	if err := s.loadManifests(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) loadManifests() error {
	loader := manifest.NewLoader(".")

	var manifests []manifest.Manifest
	for _, agent := range s.cfg.Agents {
		m, err := loader.Load(agent.ManifestPath)
		if err != nil {
			log.Printf("Warning: failed to load %s manifest: %v", agent.Name, err)
			continue
		}
		log.Printf("Loaded manifest: %s v%s (%d actions)",
			m.Name, m.Version, len(m.Actions))
		manifests = append(manifests, *m)
	}

	s.router = s.buildRouter(manifests)
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

	// Mount agent routes with RPC
	builder := router.NewBuilder(s.rpcClient)
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
