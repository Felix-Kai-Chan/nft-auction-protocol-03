// internal/api/service/auction.go
package service

import (
	"context"
	"errors"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"

	"offchain/internal/indexer/repository"
)

type AuctionService struct {
	db          *gorm.DB
	contractSvc *ContractService
}

func NewAuctionService(db *gorm.DB, contractSvc *ContractService) *AuctionService {
	return &AuctionService{
		db:          db,
		contractSvc: contractSvc,
	}
}

type CreateAuctionRequest struct {
	NFT         string `json:"nft"`
	TokenID     int64  `json:"token_id"`
	Duration    int64  `json:"duration"`
	MinBidUsd18 string `json:"min_bid_usd18"`
	Sender      string `json:"sender"`
}

type CreateAuctionResponse struct {
	AuctionID   string `json:"auction_id"`
	TxHash      string `json:"tx_hash"`
	Seller      string `json:"seller"`
	NFT         string `json:"nft"`
	TokenID     string `json:"token_id"`
	EndTime     string `json:"end_time"`
	MinBidUsd18 string `json:"min_bid_usd18"`
}

func (s *AuctionService) CreateAuction(ctx context.Context, req *CreateAuctionRequest) (*CreateAuctionResponse, error) {
	log.Printf("📝 Creating auction: nft=%s, tokenId=%d", req.NFT, req.TokenID)

	nftAddr := common.HexToAddress(req.NFT)
	tokenId := big.NewInt(req.TokenID)
	duration := big.NewInt(req.Duration)
	minBid, ok := new(big.Int).SetString(req.MinBidUsd18, 10)
	if !ok {
		return nil, errors.New("invalid min_bid_usd18")
	}

	tx, err := s.contractSvc.CreateAuction(ctx, nftAddr, tokenId, duration, minBid)
	if err != nil {
		return nil, err
	}

	receipt, _ := s.contractSvc.WaitForReceipt(ctx, tx.Hash())

	var auctionId string
	if receipt != nil && len(receipt.Logs) > 0 {
		for _, vLog := range receipt.Logs {
			if len(vLog.Topics) > 0 && vLog.Topics[0].Hex() == AuctionCreatedEventSig {
				auctionId = new(big.Int).SetBytes(vLog.Topics[1].Bytes()).String()
				break
			}
		}
	}
	if auctionId == "" {
		nextId, err := s.contractSvc.GetNextAuctionId(ctx)
		if err == nil {
			auctionId = new(big.Int).Sub(nextId, big.NewInt(1)).String()
		}
	}

	return &CreateAuctionResponse{
		AuctionID:   auctionId,
		TxHash:      tx.Hash().Hex(),
		Seller:      req.Sender,
		NFT:         req.NFT,
		TokenID:     strconv.FormatInt(req.TokenID, 10),
		EndTime:     time.Now().Add(time.Duration(req.Duration) * time.Second).Format(time.RFC3339),
		MinBidUsd18: req.MinBidUsd18,
	}, nil
}

type EndAuctionRequest struct {
	AuctionID string `json:"auction_id"`
	Sender    string `json:"sender"`
}

func (s *AuctionService) EndAuction(ctx context.Context, req *EndAuctionRequest) error {
	auctionId, ok := new(big.Int).SetString(req.AuctionID, 10)
	if !ok {
		return errors.New("invalid auction_id")
	}

	var auction repository.Auction
	result := s.db.WithContext(ctx).Where("auction_id = ?", req.AuctionID).First(&auction)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return errors.New("auction not found")
		}
		return result.Error
	}
	if auction.Ended {
		return errors.New("auction already ended")
	}

	tx, err := s.contractSvc.EndAuction(ctx, auctionId)
	if err != nil {
		return err
	}

	log.Printf("✅ Auction end tx: %s", tx.Hash().Hex())
	return nil
}

type AuctionDetail struct {
	repository.Auction
	Status        string `json:"status"`
	TimeRemaining int64  `json:"time_remaining"`
}

func (s *AuctionService) GetAuctionDetail(ctx context.Context, auctionID string) (*AuctionDetail, error) {
	var auction repository.Auction
	result := s.db.WithContext(ctx).Preload("Bids").Where("auction_id = ?", auctionID).First(&auction)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("auction not found")
		}
		return nil, result.Error
	}

	now := time.Now().Unix()
	endTime, _ := new(big.Int).SetString(auction.EndTime, 10)

	status := "active"
	if auction.Ended {
		status = "ended"
	} else if endTime != nil && now >= endTime.Int64() {
		status = "ended"
	}

	return &AuctionDetail{
		Auction:       auction,
		Status:        status,
		TimeRemaining: endTime.Int64() - now,
	}, nil
}

type ListAuctionsRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Status   string `json:"status"`
	Seller   string `json:"seller"`
	NFT      string `json:"nft"`
}

type ListAuctionsResponse struct {
	Data      []AuctionDetail `json:"data"`
	Page      int             `json:"page"`
	PageSize  int             `json:"page_size"`
	Total     int64           `json:"total"`
	TotalPage int64           `json:"total_page"`
}

func (s *AuctionService) ListAuctions(ctx context.Context, req *ListAuctionsRequest) (*ListAuctionsResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	query := s.db.WithContext(ctx).Model(&repository.Auction{})

	if req.Seller != "" {
		query = query.Where("seller = ?", req.Seller)
	}
	if req.NFT != "" {
		query = query.Where("nft = ?", req.NFT)
	}
	if req.Status == "active" {
		query = query.Where("ended = ?", false)
	} else if req.Status == "ended" {
		query = query.Where("ended = ?", true)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var auctions []repository.Auction
	offset := (req.Page - 1) * req.PageSize
	if err := query.Preload("Bids").Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&auctions).Error; err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	data := make([]AuctionDetail, len(auctions))
	for i, a := range auctions {
		endTime, _ := new(big.Int).SetString(a.EndTime, 10)
		status := "active"
		if a.Ended {
			status = "ended"
		} else if endTime != nil && now >= endTime.Int64() {
			status = "ended"
		}
		data[i] = AuctionDetail{
			Auction:       a,
			Status:        status,
			TimeRemaining: endTime.Int64() - now,
		}
	}

	return &ListAuctionsResponse{
		Data:      data,
		Page:      req.Page,
		PageSize:  req.PageSize,
		Total:     total,
		TotalPage: (total + int64(req.PageSize) - 1) / int64(req.PageSize),
	}, nil
}
