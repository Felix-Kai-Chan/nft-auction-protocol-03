// cmd/indexer/main.go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"offchain/internal/config"
	"offchain/internal/indexer"
	"offchain/internal/indexer/repository"
	"offchain/internal/infra/eth"
)

func main() {
	cfg := config.Load()

	// 设置 slog 输出格式为 JSON（生产环境推荐）
	// 如果希望保持文本格式，注释掉下面这行
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	slog.Info("Starting Indexer",
		"rpc_url", cfg.RPCURL,
		"contract_addr", cfg.ContractAddr,
		"chain_id", cfg.ChainID,
	)

	ethClient := eth.NewClient(cfg.RPCURL, cfg.ContractAddr, cfg.PrivateKey, cfg.ChainID)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	repo, err := repository.NewRepository(dsn)
	if err != nil {
		slog.Error("Failed to create repository", "error", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, err := repo.GetDB().DB()
		if err != nil {
			slog.Warn("Failed to close DB", "error", err)
		} else {
			sqlDB.Close()
		}
	}()

	listener := indexer.NewListener(ethClient, repo, cfg.ContractAddr, cfg.ChainID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutting down...")
		cancel()
	}()

	listener.Start(ctx)
}
