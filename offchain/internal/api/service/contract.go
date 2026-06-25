// internal/api/service/contract.go
package service

import (
	"context"
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"offchain/internal/contract"
	"offchain/internal/infra/eth"
)

// AuctionCreatedEventSig 事件签名
const AuctionCreatedEventSig = "0x06b9e486c68303eb64052e0493f906f3d93a1b7149b6b8dcff221aebd16c3513"

// ContractService 链上合约交互服务
type ContractService struct {
	client       *ethclient.Client
	auth         *bind.TransactOpts
	contract     *contract.Contract
	contractAddr common.Address
}

// NewContractService 创建合约服务
func NewContractService(ethClient *eth.Client, contractAddr string) (*ContractService, error) {
	addr := common.HexToAddress(contractAddr)

	instance, err := contract.NewContract(addr, ethClient.EthClient)
	if err != nil {
		return nil, err
	}

	return &ContractService{
		client:       ethClient.EthClient,
		auth:         ethClient.Auth,
		contract:     instance,
		contractAddr: addr,
	}, nil
}

// CreateAuction 创建拍卖
func (s *ContractService) CreateAuction(
	ctx context.Context,
	nft common.Address,
	tokenId *big.Int,
	duration *big.Int,
	minBidUsd18 *big.Int,
) (*types.Transaction, error) {
	log.Printf("🔨 Creating auction: nft=%s, tokenId=%s, duration=%s, minBid=%s",
		nft.Hex(), tokenId.String(), duration.String(), minBidUsd18.String())

	tx, err := s.contract.CreateAuction(s.auth, nft, tokenId, duration, minBidUsd18)
	if err != nil {
		return nil, err
	}

	log.Printf("✅ Auction created, tx: %s", tx.Hash().Hex())
	return tx, nil
}

// BidEth ETH 出价
func (s *ContractService) BidEth(ctx context.Context, auctionId *big.Int, value *big.Int) (*types.Transaction, error) {
	log.Printf("💰 Placing ETH bid: auctionId=%s, value=%s", auctionId.String(), value.String())

	auth := *s.auth
	auth.Value = value

	tx, err := s.contract.BidEth(&auth, auctionId)
	if err != nil {
		return nil, err
	}

	log.Printf("✅ ETH bid placed, tx: %s", tx.Hash().Hex())
	return tx, nil
}

// BidUSDC USDC 出价
func (s *ContractService) BidUSDC(ctx context.Context, auctionId *big.Int, amount *big.Int) (*types.Transaction, error) {
	log.Printf("💰 Placing USDC bid: auctionId=%s, amount=%s", auctionId.String(), amount.String())

	tx, err := s.contract.BidUSDC(s.auth, auctionId, amount)
	if err != nil {
		return nil, err
	}

	log.Printf("✅ USDC bid placed, tx: %s", tx.Hash().Hex())
	return tx, nil
}

// EndAuction 结束拍卖
func (s *ContractService) EndAuction(ctx context.Context, auctionId *big.Int) (*types.Transaction, error) {
	log.Printf("⏰ Ending auction: auctionId=%s", auctionId.String())

	tx, err := s.contract.EndAuction(s.auth, auctionId)
	if err != nil {
		return nil, err
	}

	log.Printf("✅ Auction ended, tx: %s", tx.Hash().Hex())
	return tx, nil
}

// GetAuction 查询拍卖信息
func (s *ContractService) GetAuction(ctx context.Context, auctionId *big.Int) (*struct {
	Seller           common.Address
	Nft              common.Address
	TokenId          *big.Int
	EndTime          *big.Int
	Ended            bool
	HighestBidToken  common.Address
	HighestBidAmount *big.Int
	HighestBidUsd18  *big.Int
	HighestBidder    common.Address
	MinBidUsd18      *big.Int
}, error) {
	auction, err := s.contract.Auctions(nil, auctionId)
	if err != nil {
		return nil, err
	}
	return &auction, nil
}

// GetNextAuctionId 获取下一个拍卖 ID
func (s *ContractService) GetNextAuctionId(ctx context.Context) (*big.Int, error) {
	return s.contract.NextAuctionId(nil)
}

// WaitForReceipt 等待交易确认（轮询）
func (s *ContractService) WaitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	for i := 0; i < 30; i++ {
		receipt, err := s.client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}
		if err.Error() != "not found" {
			return nil, err
		}
		time.Sleep(1 * time.Second)
	}
	return nil, errors.New("timeout waiting for receipt")
}
