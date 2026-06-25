// internal/api/handler/router.go
package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"offchain/internal/api/middleware"
	"offchain/internal/api/service"
)

func NewRouterWithServices(
	db *gorm.DB,
	auctionService *service.AuctionService,
	bidService *service.BidService,
) *gin.Engine {
	auctionHandler := NewAuctionHandler(db, auctionService)
	bidHandler := NewBidHandler(db, bidService)

	r := gin.Default()

	// 全局中间件
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// 健康检查
	r.GET("/health", HealthCheck)

	// 拍卖路由
	r.GET("/api/auctions", auctionHandler.GetAuctions)
	r.POST("/api/auctions", auctionHandler.CreateAuctionHandler)
	r.GET("/api/auctions/:id", auctionHandler.GetAuction)
	r.POST("/api/auctions/:id/end", auctionHandler.EndAuctionHandler)
	r.GET("/api/auctions/:id/bids", auctionHandler.GetAuctionBids)

	// 出价路由
	r.GET("/api/bids", bidHandler.GetBids)
	r.GET("/api/bids/stats", bidHandler.GetBidStats)
	r.POST("/api/bids", bidHandler.PlaceBidHandler)

	return r
}
