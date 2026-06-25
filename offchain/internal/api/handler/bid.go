// internal/api/handler/bid.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"offchain/internal/api/service"
	"offchain/internal/indexer/repository"
)

type BidHandler struct {
	db         *gorm.DB
	bidService *service.BidService
}

func NewBidHandler(db *gorm.DB, bidService *service.BidService) *BidHandler {
	return &BidHandler{
		db:         db,
		bidService: bidService,
	}
}

func (h *BidHandler) GetBids(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 {
		pageSize = 20
	}

	var bids []repository.Bid
	var total int64

	query := h.db.Model(&repository.Bid{})
	if auctionID := c.Query("auction_id"); auctionID != "" {
		query = query.Where("auction_id = ?", auctionID)
	}
	if bidder := c.Query("bidder"); bidder != "" {
		query = query.Where("bidder = ?", bidder)
	}

	query.Count(&total)
	query.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&bids)

	c.JSON(http.StatusOK, gin.H{
		"data":       bids,
		"page":       page,
		"page_size":  pageSize,
		"total":      total,
		"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (h *BidHandler) PlaceBidHandler(c *gin.Context) {
	var req service.PlaceBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	resp, err := h.bidService.PlaceBid(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *BidHandler) GetBidStats(c *gin.Context) {
	var totalBids int64
	var totalVolume string

	h.db.Model(&repository.Bid{}).Count(&totalBids)
	h.db.Model(&repository.Bid{}).Select("SUM(CAST(bid_amount AS DECIMAL(65,0))) as total").Scan(&totalVolume)

	c.JSON(http.StatusOK, gin.H{
		"total_bids":   totalBids,
		"total_volume": totalVolume,
	})
}
