// cmd/api/main.go
package main

import (
	"fmt"
	"log"

	"offchain/internal/api/handler"
	"offchain/internal/api/service"
	"offchain/internal/config"
	"offchain/internal/infra/eth"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect DB: %v", err)
	}

	ethClient := eth.NewClient(cfg.RPCURL, cfg.ContractAddr, cfg.PrivateKey, cfg.ChainID)

	contractSvc, err := service.NewContractService(ethClient, cfg.ContractAddr)
	if err != nil {
		log.Fatalf("Failed to create contract service: %v", err)
	}

	auctionService := service.NewAuctionService(db, contractSvc)
	bidService := service.NewBidService(db, contractSvc)

	router := handler.NewRouterWithServices(db, auctionService, bidService)

	log.Printf("🚀 API server starting on http://localhost:%s", cfg.APIPort)
	if err := router.Run(":" + cfg.APIPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
