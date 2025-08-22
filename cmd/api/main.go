package main

import (
	"context"
	"grls/internal/app"
	"grls/internal/config"
	"grls/internal/infrastructure/cache"
	"grls/internal/infrastructure/db"
	"grls/pkg/graceful"
	"grls/pkg/logger"

	"github.com/jpillora/overseer"
	"github.com/jpillora/overseer/fetcher"
)

func main() {
	debug := config.GetAppEnv() == "development"

	overseer.Run(overseer.Config{
		Program:       program,
		Address:       ":" + config.GetAppPort(),
		Fetcher:       &fetcher.File{Path: config.GetAppBinFile(), Interval: 5},
		Debug:         debug,
		RestartSignal: graceful.RestartSignal,
	})
}

func program(state overseer.State) {
	// Setup context with cancellation for graceful shutdown
	// This will be triggered by OS signal or overseer restarts
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // prevent potential leak

	graceful.SetupGracefulShutdown(cancel)

	cfg := config.Load()

	// Logging
	logger.InitLogFile(cfg.App.LogFilePath)

	// DB Write
	dbWriteConn, err := db.ConnectDBWrite(cfg.DB)
	if err != nil {
		errorDetails := err.Error()
		logger.WriteLogToFile("failed", "db.ConnectDBWrite", map[string]any{
			"db_name": cfg.DB.DBWrite.Name,
			"db_host": cfg.DB.DBWrite.Host,
		}, &errorDetails)
		logger.Fatal("❌ Failed to connect database write: " + errorDetails)
	}

	logger.Infof("✅ Connected to database write: %s", cfg.DB.DBWrite.Name)

	// DB Read
	dbReadConn, err := db.ConnectDBRead(cfg.DB)
	if err != nil {
		errorDetails := err.Error()
		logger.WriteLogToFile("failed", "db.ConnectDBRead", map[string]any{
			"db_name": cfg.DB.DBRead.Name,
			"db_host": cfg.DB.DBRead.Host,
		}, &errorDetails)
		logger.Fatal("❌ Failed to connect database read: " + errorDetails)
	}
	logger.WriteLogToFile("success", "db.ConnectDBRead", map[string]any{
		"db_name": cfg.DB.DBRead.Name,
		"db_host": cfg.DB.DBRead.Host,
	}, nil)
	logger.Infof("✅ Connected to database read: %s", cfg.DB.DBWrite.Name)

	// Cache
	rdb, err := cache.New(ctx, *cfg.Redis)
	if err != nil {
		errorDetails := err.Error()
		logger.WriteLogToFile("failed", "cache.New", map[string]any{}, &errorDetails)
		logger.Fatal("❌ Failed to connect cache redis : " + errorDetails)
	}
	logger.WriteLogToFile("success", "cache.New", map[string]any{}, nil)
	logger.Infof("✅ Connected to cache redis")

	// Start App
	application := app.NewApp(cfg, dbWriteConn, dbReadConn, &rdb)
	application.Start(state.Listener)

	// Block until terminated
	<-ctx.Done()

	// Graceful shutdown
	db.CloseDBWrite()
	db.CloseDBRead()
	rdb.Close()
	logger.Info("🛑 Shutting down gracefully...")
	logger.Info("✅ Cleanup done. Exiting.")
}
