package main

import (
	"context"
	"net"
	"sync"
	"time"

	"grls/internal/config"
	"grls/internal/infrastructure/cache"
	"grls/internal/infrastructure/db"
	"grls/internal/modules/wallet/store"
	"grls/pkg/graceful"
	"grls/pkg/logger"

	walletservice "grls/internal/grpc"

	"github.com/jpillora/overseer"
	"github.com/jpillora/overseer/fetcher"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	debug := config.GetAppEnv() == "development"

	overseer.Run(overseer.Config{
		Program:       program,
		Address:       ":" + config.GetAppPort(), // overseer buka 1 listener
		Fetcher:       &fetcher.File{Path: config.GetAppBinFile(), Interval: 5},
		Debug:         debug,
		RestartSignal: graceful.RestartSignal,
	})
}

func program(state overseer.State) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	graceful.SetupGracefulShutdown(cancel)

	cfg := config.Load()
	logger.InitLogFile(cfg.App.LogFilePath)

	// --- DB connections ---
	_, err := db.ConnectDBWrite(cfg.DB)
	if err != nil {
		logger.Fatal("❌ Failed DB write: " + err.Error())
	}
	_, err = db.ConnectDBRead(cfg.DB)
	if err != nil {
		logger.Fatal("❌ Failed DB read: " + err.Error())
	}
	logger.Infof("✅ DB connected (write=%s, read=%s)", cfg.DB.DBWrite.Name, cfg.DB.DBRead.Name)

	// --- Redis / Cache ---
	rdb, err := cache.New(ctx, *cfg.Redis)
	if err != nil {
		logger.Fatal("❌ Failed Redis: " + err.Error())
	}
	logger.Info("✅ Redis connected")

	var wg sync.WaitGroup

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		startGRPCServer(ctx, rdb, state.Listener)
	}()

	// Block sampai ada signal cancel
	<-ctx.Done()

	db.CloseDBWrite()
	db.CloseDBRead()
	_ = rdb.Close()

	logger.Info("🛑 Waiting for all shutdown gracefully...")
	wg.Wait()
	logger.Info("✅ Cleanup done. Exiting.")
}

// startGRPCServer men-serve gRPC di listener yang diberikan, dengan health & reflection.
// Ia akan berhenti gracefull saat ctx.Done().
func startGRPCServer(ctx context.Context, rdb redis.UniversalClient, listener net.Listener) {
	s := grpc.NewServer()

	rds := store.NewRedisWalletStore(rdb)
	walletservice.RegisterWalletService(s, rds)

	// 🔧 Register health service
	healthServer := health.NewServer()
	// Nama service sesuai dengan yang diregister di protobuf
	healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("log.LogService", healthgrpc.HealthCheckResponse_SERVING)
	healthgrpc.RegisterHealthServer(s, healthServer)
	// Enable reflection
	reflection.Register(s)

	// 🔧 Register reflection service on gRPC server
	errChan := make(chan error, 1)

	go func() {
		logger.Info("🚀 gRPC server starting...")
		errChan <- s.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		logger.Info("🔴 Stopping gRPC server...")
		time.Sleep(2 * time.Second) // Wait for 3 seconds before stopping
		logger.Info("✅ gRPC server stopped.")
	case err := <-errChan:
		if err != nil {
			logger.Error("❌ gRPC server error: " + err.Error())
		}
	}
}
