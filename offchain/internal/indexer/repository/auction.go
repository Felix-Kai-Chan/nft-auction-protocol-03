// internal/indexer/repository/auction.go
package repository

import (
	"context"
	"log"

	"gorm.io/gorm"
)

// SaveAuction 创建新的拍卖记录
func (r *Repository) SaveAuction(ctx context.Context, auction *Auction) error {
	result := r.db.WithContext(ctx).
		Where(Auction{AuctionID: auction.AuctionID}).
		Assign(auction).
		FirstOrCreate(auction)

	if result.Error != nil {
		log.Printf("SaveAuction error: %v", result.Error)
		return result.Error
	}

	log.Printf("SaveAuction success: auctionId=%s", auction.AuctionID)
	return nil
}

// GetAuction 获取拍卖信息
func (r *Repository) GetAuction(ctx context.Context, auctionID string) (*Auction, error) {
	var auction Auction
	result := r.db.WithContext(ctx).
		Where("auction_id = ?", auctionID).
		First(&auction)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}

	return &auction, nil
}
