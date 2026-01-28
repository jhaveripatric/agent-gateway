# Agent Gateway

HTTP gateway that translates REST API calls to CloudEvents for the agent ecosystem.

## Quick Start

```bash
go run ./cmd/agent-gateway
```

## Configuration

Edit `config.yaml` to configure:
- Gateway port and CORS settings
- RabbitMQ connection
- Agent manifests to load

## Endpoints

| Endpoint | Description |
|----------|-------------|
| GET /healthz | Health check |
| GET /readyz | Readiness check |
| POST /api/auth/login | Login (session-agent) |
| POST /api/auth/validate | Validate token (session-agent) |
| POST /api/auth/logout | Logout (session-agent) |

## Development

Routes are generated from agent manifests at startup. Each agent's `agent.yaml` defines:
- HTTP method and path
- Request/response event mappings
- Authentication requirements
- Rate limiting rules

## Phases

- [x] Phase 1: Core gateway with middleware
- [x] Phase 2: Manifest loading and route generation
- [ ] Phase 3: RPC client (RabbitMQ integration)
- [ ] Phase 4: JWT authentication
- [ ] Phase 5: RBAC & rate limiting
- [ ] Phase 6: Hardening & testing
