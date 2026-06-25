// internal/indexer/listener.go
package indexer

import (
	"context"
	"log/slog"
	"math/big"
	"time"

	"offchain/internal/indexer/repository"
	"offchain/internal/infra/eth"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// 事件签名常量
const (
	AuctionCreatedEventSig   = "0x06b9e486c68303eb64052e0493f906f3d93a1b7149b6b8dcff221aebd16c3513"
	BidPlacedEventSig        = "0x2808decb743a25d04efe1bd3dc192acde3be644e2f6ad1dce5d3c46643e1c602"
	AuctionEndedEventSig     = "0x596165d0521c3cb4157fad2621686f086daed4663acb3d03441a92b9277f5683"
	AuctionCancelledEventSig = "0x2809c7e17bf978fbc7194c0a694b638c4215e9140cacc6c38ca36010b45697df"
)

type Listener struct {
	ethClient    *eth.Client
	repo         *repository.Repository
	contractAddr common.Address
	chainID      int64
}

func NewListener(ethClient *eth.Client, repo *repository.Repository, contractAddr string, chainID int64) *Listener {
	return &Listener{
		ethClient:    ethClient,
		repo:         repo,
		contractAddr: common.HexToAddress(contractAddr),
		chainID:      chainID,
	}
}

// Start 开始监听链上事件（轮询方式 + 游标持久化）
func (l *Listener) Start(ctx context.Context) {
	slog.Info("Indexer started listening for events", "mode", "polling")

	client := l.ethClient.EthClient

	// ✅ 读取游标
	lastBlock, err := l.repo.GetCursor(ctx, l.contractAddr.Hex(), l.chainID)
	if err != nil {
		slog.Warn("Failed to get cursor, starting from 0", "error", err)
		lastBlock = 0
	} else if lastBlock > 0 {
		slog.Info("Resuming from block", "block", lastBlock)
	} else {
		slog.Info("No cursor found, starting from latest block")
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Indexer stopped")
			return
		case <-ticker.C:
			header, err := client.HeaderByNumber(ctx, nil)
			if err != nil {
				slog.Warn("Failed to get header", "error", err)
				continue
			}

			currentBlock := header.Number.Uint64()
			if currentBlock <= lastBlock {
				continue
			}

			// Alchemy 免费版限制：一次最多查 10 个区块
			toBlock := currentBlock
			if toBlock-lastBlock > 10 {
				toBlock = lastBlock + 10
			}

			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(lastBlock + 1)),
				ToBlock:   big.NewInt(int64(toBlock)),
				Addresses: []common.Address{l.contractAddr},
			}

			logs, err := client.FilterLogs(ctx, query)
			if err != nil {
				slog.Warn("Failed to filter logs", "error", err)
				continue
			}

			for _, vLog := range logs {
				l.handleLog(ctx, vLog)
			}

			// ✅ 保存游标
			lastBlock = toBlock
			if err := l.repo.SaveCursor(ctx, l.contractAddr.Hex(), l.chainID, lastBlock); err != nil {
				slog.Warn("Failed to save cursor", "error", err)
			}
		}
	}
}

// handleLog 分发事件给不同的处理函数
func (l *Listener) handleLog(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) == 0 {
		return
	}

	eventSig := vLog.Topics[0].Hex()

	switch eventSig {
	case AuctionCreatedEventSig:
		l.handleAuctionCreated(ctx, vLog)
	case BidPlacedEventSig:
		l.handleBidPlaced(ctx, vLog)
	case AuctionEndedEventSig:
		l.handleAuctionEnded(ctx, vLog)
	case AuctionCancelledEventSig:
		l.handleAuctionCancelled(ctx, vLog)
	default:
		// 忽略系统事件
	}
}

// handleAuctionCreated 处理 AuctionCreated 事件
func (l *Listener) handleAuctionCreated(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) < 4 {
		slog.Warn("Invalid AuctionCreated event", "topics", len(vLog.Topics)-1, "expected", 3)
		return
	}

	auctionId := new(big.Int).SetBytes(vLog.Topics[1].Bytes())
	seller := common.BytesToAddress(vLog.Topics[2].Bytes())
	nft := common.BytesToAddress(vLog.Topics[3].Bytes())

	if len(vLog.Data) < 96 {
		slog.Warn("Invalid AuctionCreated event data length", "got", len(vLog.Data), "expected", 96)
		return
	}

	tokenId := new(big.Int).SetBytes(vLog.Data[0:32])
	endTime := new(big.Int).SetBytes(vLog.Data[32:64])
	minBidUsd18 := new(big.Int).SetBytes(vLog.Data[64:96])

	slog.Info("AuctionCreated",
		"auction_id", auctionId.String(),
		"seller", seller.Hex(),
		"nft", nft.Hex(),
		"token_id", tokenId.String(),
		"end_time", endTime.String(),
		"min_bid_usd18", minBidUsd18.String(),
	)

	auction := &repository.Auction{
		AuctionID:     auctionId.String(),
		Seller:        seller.Hex(),
		NFT:           nft.Hex(),
		TokenID:       tokenId.String(),
		EndTime:       endTime.String(),
		MinBidUsd18:   minBidUsd18.String(),
		HighestBid:    "0",
		HighestBidder: "",
		Ended:         false,
	}

	if err := l.repo.SaveAuction(ctx, auction); err != nil {
		slog.Error("Failed to save auction", "error", err)
	}
}

// handleBidPlaced 处理 BidPlaced 事件
func (l *Listener) handleBidPlaced(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) < 3 {
		slog.Warn("Invalid BidPlaced event", "topics", len(vLog.Topics)-1, "expected", 2)
		return
	}

	auctionId := new(big.Int).SetBytes(vLog.Topics[1].Bytes())
	bidder := common.BytesToAddress(vLog.Topics[2].Bytes())

	if len(vLog.Data) < 96 {
		slog.Warn("Invalid BidPlaced event data length", "got", len(vLog.Data), "expected", 96)
		return
	}

	bidToken := common.BytesToAddress(vLog.Data[0:32])
	bidAmount := new(big.Int).SetBytes(vLog.Data[32:64])
	bidUsd18 := new(big.Int).SetBytes(vLog.Data[64:96])

	slog.Info("BidPlaced",
		"auction_id", auctionId.String(),
		"bidder", bidder.Hex(),
		"bid_token", bidToken.Hex(),
		"bid_amount", bidAmount.String(),
		"bid_usd18", bidUsd18.String(),
	)

	bid := &repository.Bid{
		AuctionID:   auctionId.String(),
		Bidder:      bidder.Hex(),
		BidToken:    bidToken.Hex(),
		BidAmount:   bidAmount.String(),
		BidUsd18:    bidUsd18.String(),
		BlockNumber: vLog.BlockNumber,
		TxHash:      vLog.TxHash.Hex(),
	}

	if err := l.repo.UpdateBid(ctx, bid); err != nil {
		slog.Error("Failed to save bid", "error", err)
	}
}

// handleAuctionEnded 处理 AuctionEnded 事件
func (l *Listener) handleAuctionEnded(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) < 2 {
		slog.Warn("Invalid AuctionEnded event", "topics", len(vLog.Topics)-1, "expected", 1)
		return
	}

	auctionId := new(big.Int).SetBytes(vLog.Topics[1].Bytes())

	if len(vLog.Data) < 128 {
		slog.Warn("Invalid AuctionEnded event data length", "got", len(vLog.Data), "expected", 128)
		return
	}

	winner := common.BytesToAddress(vLog.Data[0:32])
	payToken := common.BytesToAddress(vLog.Data[32:64])
	payAmount := new(big.Int).SetBytes(vLog.Data[64:96])
	payUsd18 := new(big.Int).SetBytes(vLog.Data[96:128])

	slog.Info("AuctionEnded",
		"auction_id", auctionId.String(),
		"winner", winner.Hex(),
		"pay_token", payToken.Hex(),
		"pay_amount", payAmount.String(),
		"pay_usd18", payUsd18.String(),
	)

	if err := l.repo.EndAuction(ctx, auctionId.String()); err != nil {
		slog.Error("Failed to end auction", "error", err)
	}
}

// handleAuctionCancelled 处理 AuctionCancelled 事件
func (l *Listener) handleAuctionCancelled(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) < 2 {
		slog.Warn("Invalid AuctionCancelled event", "topics", len(vLog.Topics)-1, "expected", 1)
		return
	}

	auctionId := new(big.Int).SetBytes(vLog.Topics[1].Bytes())
	slog.Info("AuctionCancelled", "auction_id", auctionId.String())

	if err := l.repo.EndAuction(ctx, auctionId.String()); err != nil {
		slog.Error("Failed to cancel auction", "error", err)
	}
}
