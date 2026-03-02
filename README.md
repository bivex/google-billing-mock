# Google Play Billing Mock Server

A local HTTP server that emulates the **Google Play Android Publisher API v3** billing endpoints. Designed for integration testing of in-app purchases and subscriptions without touching real Google Play infrastructure.

## Features

- Full coverage of all **18 IAP-relevant billing endpoints** (v1 + v2)
- **Scenario-based token matching** — token prefix determines the response shape
- **Chaos engineering** — configurable latency injection and random error rate
- **Admin API** — manage scenarios and seed purchases at runtime
- **Prometheus metrics** + structured JSON logging (Zap)
- Wire-compatible responses with the real Google Play API (same field names, types, and null semantics)

## Quick Start

### Run locally

```bash
git clone https://github.com/bivex/google-billing-mock
cd playmock
make run
```

Server starts on `http://localhost:8080`.

### Run with Docker

```bash
make docker-build
docker-compose -f deploy/docker-compose.yml up
```

### Run tests

```bash
make test        # unit tests
make test-race   # with race detector
make cover       # coverage report
```

---

## API Endpoints

All endpoints mirror the real `https://androidpublisher.googleapis.com/androidpublisher/v3` base path.

### Subscriptions v1

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/androidpublisher/v3/applications/{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}` | Get subscription purchase |
| `POST` | `.../tokens/{token}:acknowledge` | Acknowledge → **204** |
| `POST` | `.../tokens/{token}:cancel` | Cancel subscription |
| `POST` | `.../tokens/{token}:defer` | Defer expiry → `{"newExpiryTimeMillis":"..."}` |
| `POST` | `.../tokens/{token}:refund` | Refund |
| `POST` | `.../tokens/{token}:revoke` | Revoke access |

### Subscriptions v2

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `.../purchases/subscriptionsv2/tokens/{token}` | Get → `SubscriptionPurchaseV2` (lineItems[], subscriptionState) |
| `POST` | `.../subscriptionsv2/tokens/{token}:cancel` | Cancel → `{}` |
| `POST` | `.../subscriptionsv2/tokens/{token}:defer` | Defer → `ItemExpiryTimeDetails[]` |
| `POST` | `.../subscriptionsv2/tokens/{token}:revoke` | Revoke → `{}` |

### Products v1

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `.../purchases/products/{productId}/tokens/{token}` | Get product purchase |
| `POST` | `.../tokens/{token}:acknowledge` | Acknowledge → **204** |
| `POST` | `.../tokens/{token}:consume` | Consume → **204** |

### Products v2

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `.../purchases/productsv2/tokens/{token}` | Get → `ProductPurchaseV2` (purchaseStateContext) |

### Orders & Voided Purchases

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `.../orders/{orderId}` | Get order (GPA.* prefix required) |
| `GET` | `.../orders:batchGet?orderIds=...` | Batch get orders |
| `POST` | `.../orders/{orderId}:refund` | Refund order → **204** |
| `GET` | `.../purchases/voidedpurchases` | List voided purchases (empty) |

### Utility

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness probe → `{"status":"ok"}` |
| `GET` | `/ready` | Readiness probe |
| `GET` | `/metrics` | Prometheus metrics |

### Admin API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/admin/scenarios` | List loaded scenarios |
| `POST` | `/admin/scenarios` | Add scenario |
| `DELETE` | `/admin/scenarios/{name}` | Remove scenario |
| `POST` | `/admin/scenarios/reload` | Reload from file |
| `POST` | `/admin/purchases/subscriptions` | Seed a subscription |
| `GET` | `/admin/purchases/subscriptions` | List all subscriptions |
| `POST` | `/admin/purchases/products` | Seed a product purchase |
| `GET` | `/admin/purchases/products` | List all product purchases |

---

## Scenario-Based Token Matching

The mock resolves purchase state by matching the **token prefix** against configured scenarios. No real token validation is performed.

### Built-in scenarios

| Scenario | Token prefix | Type | Behaviour |
|----------|-------------|------|-----------|
| `valid_active` | `valid_active` | subscription | Active, auto-renewing, acknowledged |
| `expired_no_renew` | `expired` | subscription | Expired, `autoRenewing=false` |
| `canceled_user` | `canceled` | subscription | `cancelReason=0` (user) |
| `pending_payment` | `pending` | subscription | `paymentState=0` (grace period) |
| `invalid_token` | `invalid` | subscription | **Forced 410** `PURCHASE_TOKEN_EXPIRED` |
| `valid_product` | `product_valid` | product | Purchased + acknowledged |
| `pending_product` | `product_pending` | product | Unconsumed, unacknowledged |

### Example

```bash
# Returns active subscription (matches "valid_active" prefix)
curl http://localhost:8080/androidpublisher/v3/applications/com.example.app \
  /purchases/subscriptions/com.example.sub/tokens/valid_active_user_123

# Returns 410 error (matches "invalid" prefix)
curl http://localhost:8080/androidpublisher/v3/applications/com.example.app \
  /purchases/subscriptions/com.example.sub/tokens/invalid_token_abc
```

### Custom scenario (Admin API)

```bash
curl -X POST http://localhost:8080/admin/scenarios \
  -H "Content-Type: application/json" \
  -d '{
    "name": "grace_period",
    "type": "subscription",
    "token_prefix": "grace_",
    "purchase_state": 0,
    "payment_state": 0,
    "auto_renewing": true,
    "acknowledgement_state": 1,
    "expiry_time_millis": 9999999999000
  }'
```

---

## Chaos Engineering

Inject failures per-server or per-request.

### Server-wide (config)

```yaml
mock:
  default_latency_ms: 200   # add 200 ms to every response
  error_rate: 0.05          # 5% of requests return a random 5xx
```

### Per-request (headers)

```bash
# Force 500 ms latency on this request
curl -H "X-Mock-Latency-Ms: 500" http://localhost:8080/...

# Force 100% error rate on this request
curl -H "X-Mock-Error-Rate: 1.0" http://localhost:8080/...
```

---

## Configuration

Config is loaded from YAML, with environment variable overrides.

```yaml
# config/default.yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 10s

mock:
  scenarios_path: config/scenarios/default.json
  default_latency_ms: 0
  error_rate: 0.0

log:
  level: info          # debug | info | warn | error

metrics:
  enabled: true
  path: /metrics
```

### Environment variables

All keys map to `MOCK_<SECTION>_<KEY>` (uppercase, underscores):

```bash
MOCK_SERVER_PORT=9090
MOCK_LOG_LEVEL=debug
MOCK_MOCK_DEFAULT_LATENCY_MS=100
MOCK_MOCK_ERROR_RATE=0.1
MOCK_MOCK_SCENARIOS_PATH=/etc/mock/scenarios.json
```

Pass a custom config file with the `-config` flag:

```bash
./server -config /etc/mock/config.yaml
```

---

## Project Structure

```
.
├── cmd/server/          # Entrypoint, dependency wiring, graceful shutdown
├── config/              # default.yaml + scenarios/default.json
├── deploy/              # Dockerfile, docker-compose.yml
├── doc/                 # Task specification
├── internal/
│   ├── application/
│   │   ├── dto/         # Request/response structs (wire format)
│   │   └── usecase/     # Business logic use cases
│   ├── domain/
│   │   ├── entity/      # Aggregates: SubscriptionPurchase, ProductPurchase
│   │   ├── event/       # Domain events
│   │   └── repository/  # Repository port interface
│   └── infrastructure/
│       ├── config/      # Viper loader
│       ├── http/
│       │   ├── handler/ # HTTP handlers (v1, v2, orders, admin, health)
│       │   ├── middleware/ # Logging, metrics, chaos, correlation-id
│       │   └── router.go
│       ├── logger/      # Zap factory
│       ├── metrics/     # Prometheus instruments
│       └── mock/        # InMemoryRepository + ScenarioManager
└── androidpublisher_v3.json  # Google Play API Discovery document
```

Architecture follows **Clean/Hexagonal** principles:
`Domain → Application → Infrastructure`. The domain has zero external dependencies.

---

## Makefile Targets

```bash
make build        # go build ./...
make run          # go run ./cmd/server
make test         # go test ./...
make test-race    # go test -race ./...
make cover        # open coverage report in browser
make lint         # golangci-lint run
make docker-build # docker build
```

---

## Differences from the Real API

| Behaviour | Real API | Mock |
|-----------|----------|------|
| Auth | OAuth2 Bearer required | None (always passes) |
| Token validity | Cryptographically verified | Prefix match only |
| Expiry | Real calendar time | Configurable via scenario |
| `orderId` format | `GPA.XXXX-XXXX-XXXX-XXXXX` | Same format, random digits |
| `voidedpurchases` | Historical data | Always empty |
| `orders.get` | Real order lookup | Synthetic response for any `GPA.*` orderId |
| v2 `:defer` body | ISO 8601 duration parsed | Extends expiry by 30 days |
