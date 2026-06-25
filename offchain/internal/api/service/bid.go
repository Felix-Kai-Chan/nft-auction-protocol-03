// internal/api/service/bid.go
package service

import (
	"context"
	"errors"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"

	"offchain/internal/indexer/repository"
)

type BidService struct {
	db          *gorm.DB
	contractSvc *ContractService
}

func NewBidService(db *gorm.DB, contractSvc *ContractService) *BidService {
	return &BidService{
		db:          db,
		contractSvc: contractSvc,
	}
}

type PlaceBidRequest struct {
	AuctionID string `json:"auction_id"`
	Bidder    string `json:"bidder"`
	Token     string `json:"token"`
	Amount    string `json:"amount"`
}

type PlaceBidResponse struct {
	TxHash    string `json:"tx_hash"`
	AuctionID string `json:"auction_id"`
	Bidder    string `json:"bidder"`
	Token     string `json:"token"`
	Amount    string `json:"amount"`
}

func (s *BidService) PlaceBid(ctx context.Context, req *PlaceBidRequest) (*PlaceBidResponse, error) {
	log.Printf("💰 Placing bid: auctionId=%s, bidder=%s, token=%s", req.AuctionID, req.Bidder, req.Token)

	var auction repository.Auction
	result := s.db.WithContext(ctx).Where("auction_id = ?", req.AuctionID).First(&auction)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("auction not found")
		}
		return nil, result.Error
	}
	if auction.Ended {
		return nil, errors.New("auction already ended")
	}

	auctionId, ok := new(big.Int).SetString(req.AuctionID, 10)
	if !ok {
		return nil, errors.New("invalid auction_id")
	}
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		return nil, errors.New("invalid amount")
	}

	var txHash common.Hash
	switch req.Token {
	case "eth":
		tx, err := s.contractSvc.BidEth(ctx, auctionId, amount)
		if err != nil {
			return nil, err
		}
		txHash = tx.Hash()
	case "usdc":
		tx, err := s.contractSvc.BidUSDC(ctx, auctionId, amount)
		if err != nil {
			return nil, err
		}
		txHash = tx.Hash()
	default:
		return nil, errors.New("unsupported token, use 'eth' or 'usdc'")
	}

	return &PlaceBidResponse{
		TxHash:    txHash.Hex(),
		AuctionID: req.AuctionID,
		Bidder:    req.Bidder,
		Token:     req.Token,
		Amount:    req.Amount,
	}, nil
}

func (s *BidService) GetBidHistory(ctx context.Context, auctionID string) ([]repository.Bid, error) {
	var bids []repository.Bid
	result := s.db.WithContext(ctx).Where("auction_id = ?", auctionID).Order("created_at DESC").Find(&bids)
	if result.Error != nil {
		return nil, result.Error
	}
	return bids, nil
}
