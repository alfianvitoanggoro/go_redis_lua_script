// cmd/server/main.go
package main

import (
	"context"
	"net"
	"sync"

	"grls/internal/async" // <-- goroutine processor FIFO
	"grls/internal/config"
	grpcserver "grls/internal/grpc"
	"grls/internal/infrastructure/cache"
	"grls/internal/infrastructure/db"
	"grls/internal/infrastructure/repository"
	"grls/internal/store" // <-- RedisQueue (Lua enqueue/release)
	"grls/pkg/graceful"
	"grls/pkg/logger"

	"github.com/jpillora/overseer"
	"github.com/jpillora/overseer/fetcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	debug := config.GetAppEnv() == "development"

	overseer.Run(overseer.Config{
		Program:       program,
		Address:       ":" + config.GetAppPort(), // overseer menyediakan listener
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
	dbWrite, err := db.ConnectDBWrite(cfg.DB)
	if err != nil {
		logger.Fatal("❌ Failed DB write: " + err.Error())
	}
	dbRead, err := db.ConnectDBRead(cfg.DB)
	if err != nil {
		logger.Fatal("❌ Failed DB read: " + err.Error())
	}
	logger.Infof("✅ DB connected (write=%s, read=%s)", cfg.DB.DBWrite.Name, cfg.DB.DBRead.Name)

	// --- Redis connection ---
	rdb, err := cache.ConnectRedis(ctx, *cfg.Redis)
	if err != nil {
		logger.Fatal("❌ Redis connect: " + err.Error())
	}
	logger.Info("✅ Redis connected")

	// --- Dependencies ---
	repo := repository.NewWalletRepository(dbWrite, dbRead)
	queue := store.NewRedisQueue(rdb) // pakai Lua enqueue/release

	// --- Start async processor (BRPOP ready:wallet -> DB -> release) ---
	proc := async.NewProcessor(rdb, repo, queue)
	go proc.Run(ctx)

	// --- Start gRPC server ---
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startGRPCServer(ctx, queue, state.Listener)
	}()

	// Block sampai ada signal cancel
	<-ctx.Done()

	// Cleanup
	db.CloseDBWrite()
	db.CloseDBRead()
	_ = rdb.Close()

	logger.Info("🛑 Waiting for all shutdown gracefully...")
	wg.Wait()
	logger.Info("✅ Cleanup done. Exiting.")
}

// startGRPCServer menjalankan gRPC di listener yang diberikan, lengkap dengan health & reflection.
// Berhenti gracefully saat ctx.Done().
func startGRPCServer(ctx context.Context, queue *store.RedisQueue, listener net.Listener) {
	s := grpc.NewServer()

	// Register wallet service (enqueue-only)
	grpcserver.RegisterWalletService(s, queue)

	// Health service
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("wallet.v1.WalletService", healthgrpc.HealthCheckResponse_SERVING)
	healthgrpc.RegisterHealthServer(s, healthServer)

	// Reflection (dev tools: grpcurl/evans)
	reflection.Register(s)

	errChan := make(chan error, 1)
	go func() {
		logger.Info("🚀 gRPC server starting...")
		errChan <- s.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		logger.Info("🔴 Stopping gRPC server...")
		s.GracefulStop()
		logger.Info("✅ gRPC server stopped.")
	case err := <-errChan:
		if err != nil {
			logger.Error("❌ gRPC server error: " + err.Error())
		}
	}
}
