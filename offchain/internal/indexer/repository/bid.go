// internal/indexer/repository/bid.go
package repository

import (
	"context"
	"log"

	"gorm.io/gorm"
)

// UpdateBid 更新拍卖的最高出价，并插入出价记录
func (r *Repository) UpdateBid(ctx context.Context, bid *Bid) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&Auction{}).
			Where("auction_id = ? AND (highest_bid < ? OR highest_bid = '0')",
				bid.AuctionID, bid.BidAmount).
			Updates(map[string]interface{}{
				"highest_bid":    bid.BidAmount,
				"highest_bidder": bid.Bidder,
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			log.Printf("UpdateBid: bid not higher than current highest for auction %s", bid.AuctionID)
		}

		if err := tx.Create(bid).Error; err != nil {
			return err
		}

		log.Printf("UpdateBid success: auctionId=%s, bidder=%s, amount=%s",
			bid.AuctionID, bid.Bidder, bid.BidAmount)
		return nil
	})
}

// EndAuction 标记拍卖结束
func (r *Repository) EndAuction(ctx context.Context, auctionID string) error {
	result := r.db.WithContext(ctx).
		Model(&Auction{}).
		Where("auction_id = ? AND ended = ?", auctionID, false).
		Update("ended", true)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		log.Printf("EndAuction: auction %s not found or already ended", auctionID)
	} else {
		log.Printf("EndAuction success: auctionId=%s", auctionID)
	}

	return nil
}
