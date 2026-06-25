// internal/indexer/repository/db.go
package repository

import (
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Repository 负责数据存储
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建 Repository 实例
func NewRepository(dsn string) (*Repository, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Auction{}, &Bid{}, &SyncCursor{}); err != nil {
		return nil, err
	}

	log.Println("Database tables migrated successfully")
	return &Repository{db: db}, nil
}

// GetDB 获取 gorm.DB 实例
func (r *Repository) GetDB() *gorm.DB {
	return r.db
}

// ========== Models ==========

// Auction 拍卖数据结构
type Auction struct {
	ID            uint      `gorm:"primaryKey"`
	AuctionID     string    `gorm:"column:auction_id;type:varchar(78);uniqueIndex;not null"`
	Seller        string    `gorm:"column:seller;type:varchar(42);not null"`
	NFT           string    `gorm:"column:nft;type:varchar(42);not null"`
	TokenID       string    `gorm:"column:token_id;type:varchar(78);not null"`
	EndTime       string    `gorm:"column:end_time;type:varchar(78);not null"`
	MinBidUsd18   string    `gorm:"column:min_bid_usd18;type:varchar(78);not null"`
	HighestBid    string    `gorm:"column:highest_bid;type:varchar(78);default:'0'"`
	HighestBidder string    `gorm:"column:highest_bidder;type:varchar(42);default:''"`
	Ended         bool      `gorm:"column:ended;default:false"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`

	Bids []Bid `gorm:"foreignKey:AuctionID;references:AuctionID"`
}

func (Auction) TableName() string { return "auctions" }

// Bid 出价数据结构
type Bid struct {
	ID          uint      `gorm:"primaryKey"`
	AuctionID   string    `gorm:"column:auction_id;type:varchar(78);not null;index"`
	Bidder      string    `gorm:"column:bidder;type:varchar(42);not null;index"`
	BidToken    string    `gorm:"column:bid_token;type:varchar(42);not null"`
	BidAmount   string    `gorm:"column:bid_amount;type:varchar(78);not null"`
	BidUsd18    string    `gorm:"column:bid_usd18;type:varchar(78);not null"`
	BlockNumber uint64    `gorm:"column:block_number;not null"`
	TxHash      string    `gorm:"column:tx_hash;type:varchar(66);not null;index"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (Bid) TableName() string { return "bids" }

// SyncCursor 索引进度
type SyncCursor struct {
	ID              uint      `gorm:"primaryKey"`
	ContractAddress string    `gorm:"column:contract_address;type:varchar(42);uniqueIndex:uk_contract_chain"`
	ChainID         int64     `gorm:"column:chain_id;uniqueIndex:uk_contract_chain"`
	LastBlock       uint64    `gorm:"column:last_block"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (SyncCursor) TableName() string { return "sync_cursors" }
