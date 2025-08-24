package main

import (
	"context"
	"net"
	"os"
	"sync"
	"time"

	"grls/internal/app/factory"
	"grls/internal/config"
	"grls/internal/infrastructure/cache"
	"grls/internal/infrastructure/db"
	"grls/internal/infrastructure/repository"
	"grls/internal/modules/wallet/store"
	"grls/internal/worker"
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
	dbWrite, err := db.ConnectDBWrite(cfg.DB)
	if err != nil {
		logger.Fatal("❌ Failed DB write: " + err.Error())
	}

	dbRead, err := db.ConnectDBRead(cfg.DB)
	if err != nil {
		logger.Fatal("❌ Failed DB read: " + err.Error())
	}
	logger.Infof("✅ DB connected (write=%s, read=%s)", cfg.DB.DBWrite.Name, cfg.DB.DBRead.Name)

	// --- Redis connections ---
	rdbAPI, err := cache.NewAPI(ctx, *cfg.Redis)
	if err != nil {
		logger.Fatal("❌ Redis API: " + err.Error())
	}

	rdbWorker, err := cache.NewWorker(ctx, *cfg.Redis, cfg.Worker.WorkerCount)
	if err != nil {
		logger.Fatal("❌ Redis Worker: " + err.Error())
	}
	logger.Infof("✅ Redis connected client and worker")

	f := factory.NewFactory(dbWrite, dbRead)

	var wg sync.WaitGroup

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		startGRPCServer(ctx, rdbAPI, state.Listener)
	}()

	// --- Start Redis→DB workers (multi instance) ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		startWorkers(ctx, rdbWorker, f.WalletRepository, cfg.Worker)
	}()

	// Block sampai ada signal cancel
	<-ctx.Done()

	db.CloseDBWrite()
	db.CloseDBRead()
	_ = rdbAPI.Close()
	_ = rdbWorker.Close()

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

// startWorkers: spawn beberapa consumer untuk stream:wallet (multi instance)
func startWorkers(ctx context.Context, rdb redis.UniversalClient, repo *repository.WalletRepository, workerConfig *config.WorkerConfig) {
	workerCount := workerConfig.WorkerCount
	host, _ := os.Hostname()

	logger.Infof("🧵 Starting %d wallet workers...", workerCount)

	var wg sync.WaitGroup
	for i := 1; i <= workerCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			consumer := worker.ConsumerName(host, i)
			w := worker.NewWalletStreamWorker(rdb, repo, &worker.Options{
				Stream:       "stream:wallet",
				Group:        "wallet_cg",
				Block:        5 * time.Second,
				Batch:        200,
				MinIdle:      30 * time.Second,
				TrimAfterAck: false, // set true jika mau XDEL setelah ACK
			})
			logger.Infof("🧵 worker started: %s", consumer)
			w.Run(ctx, consumer)
			logger.Infof("🧵 worker stopped: %s", consumer)
		}(i)
	}

	// tunggu semua worker saat ctx.Done()
	<-ctx.Done()
	wg.Wait()
}
