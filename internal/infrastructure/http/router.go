package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/bivex/google-billing-mock/internal/infrastructure/config"
	"github.com/bivex/google-billing-mock/internal/infrastructure/http/handler"
	"github.com/bivex/google-billing-mock/internal/infrastructure/http/middleware"
	"github.com/bivex/google-billing-mock/internal/infrastructure/metrics"
	"go.uber.org/zap"
)

// NewRouter wires all routes and middleware.
func NewRouter(
	cfg *config.Config,
	subH *handler.SubscriptionHandler,
	subV2H *handler.SubscriptionV2Handler,
	prodH *handler.ProductHandler,
	prodV2H *handler.ProductV2Handler,
	orderH *handler.OrderHandler,
	healthH *handler.HealthHandler,
	adminH *handler.AdminHandler,
	m *metrics.Metrics,
	logger *zap.Logger,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware stack
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CorrelationID)
	r.Use(middleware.Logging(logger))
	if cfg.Metrics.Enabled {
		r.Use(middleware.Metrics(m))
	}
	r.Use(middleware.Chaos(middleware.ChaosConfig{
		DefaultLatencyMs: cfg.Mock.DefaultLatencyMs,
		ErrorRate:        cfg.Mock.ErrorRate,
	}))

	// Health / readiness
	r.Get("/health", healthH.Health)
	r.Get("/ready", healthH.Ready)

	// Prometheus metrics
	if cfg.Metrics.Enabled {
		r.Handle(cfg.Metrics.Path, promhttp.Handler())
	}

	// ─── Google Play Developer API v3 ────────────────────────────────────────
	// Chi v5 supports regex param patterns: {param:pattern}
	// Using [^:]+ to stop token capture before the colon action suffix.
	const base = "/androidpublisher/v3/applications/{packageName}"

	// Subscriptions
	subBase := base + "/purchases/subscriptions/{subscriptionId}/tokens/{token:[^:]+}"
	r.Get(subBase, subH.Get)
	r.Post(subBase+":acknowledge", subH.Acknowledge)
	r.Post(subBase+":cancel", subH.Cancel)
	r.Post(subBase+":refund", subH.Refund)
	r.Post(subBase+":revoke", subH.Revoke)
	r.Post(subBase+":defer", subH.Defer)

	// Products
	prodBase := base + "/purchases/products/{productId}/tokens/{token:[^:]+}"
	r.Get(prodBase, prodH.Get)
	r.Post(prodBase+":acknowledge", prodH.Acknowledge)
	r.Post(prodBase+":consume", prodH.Consume)

	// Products v2 (no productId in path)
	r.Get(base+"/purchases/productsv2/tokens/{token:[^:]+}", prodV2H.Get)

	// Subscriptions v2 (no subscriptionId in path)
	subV2Base := base + "/purchases/subscriptionsv2/tokens/{token:[^:]+}"
	r.Get(subV2Base, subV2H.Get)
	r.Post(subV2Base+":cancel", subV2H.Cancel)
	r.Post(subV2Base+":revoke", subV2H.Revoke)
	r.Post(subV2Base+":defer", subV2H.Defer)

	// Voided purchases
	r.Get(base+"/purchases/voidedpurchases", orderH.VoidedPurchasesList)

	// Orders
	r.Get(base+"/orders:batchGet", orderH.BatchGet)
	r.Get(base+"/orders/{orderId:[^:]+}", orderH.Get)
	r.Post(base+"/orders/{orderId:[^:]+}:refund", orderH.Refund)

	// ─── Admin API ────────────────────────────────────────────────────────────
	r.Route("/admin", func(r chi.Router) {
		r.Get("/scenarios", adminH.ListScenarios)
		r.Post("/scenarios", adminH.AddScenario)
		r.Post("/scenarios/reload", adminH.ReloadScenarios)
		r.Delete("/scenarios/{name}", adminH.DeleteScenario)

		r.Post("/purchases/subscriptions", adminH.SeedSubscription)
		r.Get("/purchases/subscriptions", adminH.ListSubscriptions)
		r.Post("/purchases/products", adminH.SeedProduct)
		r.Get("/purchases/products", adminH.ListProducts)
	})

	return r
}
