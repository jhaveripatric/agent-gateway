package router

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jhaveripatric/agent-gateway/internal/manifest"
)

// BuildRoutes creates routes from agent manifests.
func BuildRoutes(manifests []manifest.Manifest) chi.Router {
	r := chi.NewRouter()

	for _, m := range manifests {
		for _, action := range m.Actions {
			pattern := "/api" + action.HTTP.Path
			log.Printf("Route: %s %s -> %s.%s",
				action.HTTP.Method, pattern, m.Name, action.Name)

			handler := placeholderHandler(m.Name, action)

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

func placeholderHandler(agent string, action manifest.Action) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "not_implemented",
			"agent":  agent,
			"action": action.Name,
			"event":  action.Request.Event,
		})
	}
}
