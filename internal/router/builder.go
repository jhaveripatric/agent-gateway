package router

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jhaveripatric/agent-gateway/internal/auth"
	"github.com/jhaveripatric/agent-gateway/internal/manifest"
	"github.com/jhaveripatric/agent-gateway/internal/middleware"
	"github.com/jhaveripatric/agent-gateway/internal/rpc"
)

// Builder creates routes from agent manifests.
type Builder struct {
	rpc         *rpc.Client
	jwtVerifier *auth.JWTVerifier
}

// NewBuilder creates a route builder with RPC client and JWT verifier.
func NewBuilder(rpcClient *rpc.Client, jwtVerifier *auth.JWTVerifier) *Builder {
	return &Builder{
		rpc:         rpcClient,
		jwtVerifier: jwtVerifier,
	}
}

// Build creates routes from agent manifests.
func (b *Builder) Build(manifests []manifest.Manifest) chi.Router {
	r := chi.NewRouter()

	for _, m := range manifests {
		for _, action := range m.Actions {
			pattern := "/api" + action.HTTP.Path
			authType := "none"
			if action.Auth != "" {
				authType = action.Auth
			}
			log.Printf("Route: %s %s -> %s.%s (auth: %s)",
				action.HTTP.Method, pattern, m.Name, action.Name, authType)

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

		// 1. Authentication (if required)
		var claims *auth.Claims
		if action.Auth == "bearer" {
			token, err := auth.ExtractToken(r)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid token", requestID)
				return
			}

			claims, err = b.jwtVerifier.Verify(token)
			if err != nil {
				log.Printf("[%s] JWT verification failed: %v", requestID, err)
				writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid token", requestID)
				return
			}

			ctx = auth.WithClaims(ctx, claims)
		}

		// 2. Parse request body
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

		// 3. Add auth context to event data (if authenticated)
		if claims != nil {
			data["_auth"] = map[string]any{
				"user_id":  claims.UserID,
				"username": claims.Username,
				"roles":    claims.Roles,
			}
		}

		// 4. Add client info
		data["_client_ip"] = r.RemoteAddr
		data["_request_id"] = requestID

		// 5. RPC call
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
