package router

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jhaveripatric/agent-gateway/internal/manifest"
	"github.com/jhaveripatric/agent-gateway/internal/middleware"
	"github.com/jhaveripatric/agent-gateway/internal/rpc"
)

// Builder creates routes from agent manifests.
type Builder struct {
	rpc *rpc.Client
}

// NewBuilder creates a route builder with RPC client.
func NewBuilder(rpcClient *rpc.Client) *Builder {
	return &Builder{rpc: rpcClient}
}

// Build creates routes from agent manifests.
func (b *Builder) Build(manifests []manifest.Manifest) chi.Router {
	r := chi.NewRouter()

	for _, m := range manifests {
		for _, action := range m.Actions {
			pattern := "/api" + action.HTTP.Path
			log.Printf("Route: %s %s -> %s.%s",
				action.HTTP.Method, pattern, m.Name, action.Name)

			handler := b.buildActionHandler(m, action)

			switch action.HTTP.Method {
			case "GET":
				r.Get(pattern, handler)
			case "POST":
				r.Post(pattern, handler)
			case "PUT":
				r.Put(pattern, handler)
			case "DELETE":
				r.Delete(pattern, handler)
			default:
				log.Printf("Warning: unknown method %s for %s", action.HTTP.Method, pattern)
			}
		}
	}

	return r
}

func (b *Builder) buildActionHandler(m manifest.Manifest, action manifest.Action) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		requestID := middleware.GetRequestID(ctx)

		// Parse request body
		var data map[string]any
		if r.Body != nil && r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", requestID)
				return
			}
		}
		if data == nil {
			data = make(map[string]any)
		}

		// Add client info
		data["_client_ip"] = r.RemoteAddr
		data["_request_id"] = requestID

		// RPC call
		timeout := action.Timeout
		if timeout == 0 {
			timeout = 5 * time.Second
		}

		resp, err := b.rpc.Call(ctx, action.Request.Event, data, timeout)
		if err != nil {
			if err == rpc.ErrTimeout {
				writeError(w, http.StatusGatewayTimeout, "gateway_timeout", "Agent did not respond", requestID)
				return
			}
			log.Printf("[%s] RPC error: %v", requestID, err)
			writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Agent unavailable", requestID)
			return
		}

		// Map response to HTTP status
		status := http.StatusOK
		if resp.Type == action.Response.Failure.Event {
			status = action.Response.Failure.Status
			if status == 0 {
				status = http.StatusUnauthorized
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(resp.Data)
	}
}

func writeError(w http.ResponseWriter, status int, errCode, message, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":      errCode,
		"message":    message,
		"request_id": requestID,
	})
}
