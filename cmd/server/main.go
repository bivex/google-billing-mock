package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bivex/google-billing-mock/internal/application/usecase"
	"github.com/bivex/google-billing-mock/internal/infrastructure/config"
	httpinfra "github.com/bivex/google-billing-mock/internal/infrastructure/http"
	"github.com/bivex/google-billing-mock/internal/infrastructure/http/handler"
	"github.com/bivex/google-billing-mock/internal/infrastructure/logger"
	"github.com/bivex/google-billing-mock/internal/infrastructure/metrics"
	"github.com/bivex/google-billing-mock/internal/infrastructure/mock"
	"go.uber.org/zap"
)

func main() {
	cfgFile := flag.String("config", "", "path to YAML config file (optional)")
	flag.Parse()

	cfg, err := config.Load(*cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Log.Level)
	defer log.Sync() //nolint:errcheck

	m := metrics.New()

	scenarioMgr := mock.NewScenarioManager()
	if err := scenarioMgr.LoadFromFile(cfg.Mock.ScenariosPath); err != nil {
		log.Warn("could not load scenarios file, starting with empty scenarios",
			zap.String("path", cfg.Mock.ScenariosPath),
			zap.Error(err),
		)
	}

	repo := mock.NewInMemoryRepository(scenarioMgr)

	// Wire use cases
	getSub := usecase.NewGetSubscription(repo, log)
	getSubV2 := usecase.NewGetSubscriptionV2(repo, log)
	getProd := usecase.NewGetProduct(repo, log)
	getProdV2 := usecase.NewGetProductV2(repo, log)
	ack := usecase.NewAcknowledge(repo, log)
	cancel := usecase.NewCancel(repo, log)
	refund := usecase.NewRefund(repo, log)
	revoke := usecase.NewRevoke(repo, log)
	deferSub := usecase.NewDeferSubscription(repo, log)
	deferSubV2 := usecase.NewDeferSubscriptionV2(repo, log)
	consume := usecase.NewConsumeProduct(repo, log)

	// Wire handlers
	subH := handler.NewSubscriptionHandler(getSub, ack, cancel, refund, revoke, deferSub, log)
	subV2H := handler.NewSubscriptionV2Handler(getSubV2, cancel, revoke, deferSubV2, log)
	prodH := handler.NewProductHandler(getProd, ack, consume, log)
	prodV2H := handler.NewProductV2Handler(getProdV2, log)
	orderH := handler.NewOrderHandler(log)
	healthH := handler.NewHealthHandler()
	adminH := handler.NewAdminHandler(repo, scenarioMgr, cfg.Mock.ScenariosPath, log)

	router := httpinfra.NewRouter(cfg, subH, subV2H, prodH, prodV2H, orderH, healthH, adminH, m, log)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info("mock server starting", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gracefully")
	ctx, cancelCtx := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancelCtx()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}
	log.Info("server stopped")
}
