// internal/api/handler/auction.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"offchain/internal/api/service"
	"offchain/internal/indexer/repository"
)

type AuctionHandler struct {
	db             *gorm.DB
	auctionService *service.AuctionService
}

func NewAuctionHandler(db *gorm.DB, auctionService *service.AuctionService) *AuctionHandler {
	return &AuctionHandler{
		db:             db,
		auctionService: auctionService,
	}
}

func (h *AuctionHandler) GetAuctions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize < 1 {
		pageSize = 10
	}

	req := &service.ListAuctionsRequest{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Seller:   c.Query("seller"),
		NFT:      c.Query("nft"),
	}

	resp, err := h.auctionService.ListAuctions(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AuctionHandler) GetAuction(c *gin.Context) {
	auctionID := c.Param("id")

	detail, err := h.auctionService.GetAuctionDetail(c.Request.Context(), auctionID)
	if err != nil {
		if err.Error() == "auction not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Auction not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *AuctionHandler) GetAuctionBids(c *gin.Context) {
	auctionID := c.Param("id")

	var bids []repository.Bid
	h.db.Where("auction_id = ?", auctionID).Order("created_at DESC").Find(&bids)
	c.JSON(http.StatusOK, bids)
}

func (h *AuctionHandler) CreateAuctionHandler(c *gin.Context) {
	var req service.CreateAuctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	resp, err := h.auctionService.CreateAuction(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AuctionHandler) EndAuctionHandler(c *gin.Context) {
	auctionID := c.Param("id")

	var req service.EndAuctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}
	req.AuctionID = auctionID

	if err := h.auctionService.EndAuction(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Auction ended successfully",
		"auction_id": auctionID,
	})
}
