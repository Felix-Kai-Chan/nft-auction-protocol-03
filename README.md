# 🏛️ NFT Auction Protocol

基于以太坊的 NFT 拍卖系统，包含智能合约、事件索引器（Indexer）和 RESTful API。

[![Solidity](https://img.shields.io/badge/Solidity-0.8.28-363636)](https://soliditylang.org/)
[![Go](https://img.shields.io/badge/Go-1.21-00ADD8)](https://golang.org/)
[![Gin](https://img.shields.io/badge/Gin-1.10-00ADD8)](https://gin-gonic.com/)
[![MySQL](https://img.shields.io/badge/MySQL-8.0-4479A1)](https://mysql.com/)
[![Foundry](https://img.shields.io/badge/Foundry-1.0-FF6B00)](https://book.getfoundry.sh/)

---

## 📋 目录

- [功能特性](#-功能特性)
- [技术栈](#-技术栈)
- [项目结构](#-项目结构)
- [快速开始](#-快速开始)
- [测试](#-测试)
- [部署到 Sepolia](#-部署到-sepolia)
- [API 接口](#-api-接口)
- [Docker 部署](#-docker-部署)
- [截图](#-截图)
- [许可证](#-许可证)

---

## ✨ 功能特性

### 链上合约
- ✅ 创建拍卖（支持 ETH / USDC 出价）
- ✅ 实时出价（ETH / USDC）
- ✅ 拍卖结束（自动结算 + 手续费）
- ✅ UUPS 可升级代理
- ✅ 完整的错误处理

### 事件索引器 (Indexer)
- ✅ 监听 `AuctionCreated` / `BidPlaced` / `AuctionEnded` / `AuctionCancelled`
- ✅ 轮询模式 + 游标持久化（`sync_cursors`）
- ✅ 重启从断点继续，不漏事件
- ✅ 结构化日志（JSON 格式，支持 ELK/Loki）

### RESTful API
- ✅ 查询拍卖列表（分页 + 状态筛选）
- ✅ 查询拍卖详情（含出价历史）
- ✅ 查询出价统计
- ✅ 创建拍卖（通过 API 调用链上合约）
- ✅ 出价（ETH / USDC）
- ✅ 结束拍卖

---

## 🛠 技术栈

| 层 | 技术 |
|---|---|
| **智能合约** | Solidity 0.8.28, Foundry, OpenZeppelin |
| **后端框架** | Go 1.21, Gin, GORM |
| **数据库** | MySQL 8.0 |
| **链上交互** | go-ethereum, abigen |
| **日志** | slog（Go 官方结构化日志） |

---

## 📁 项目结构
nft-auction-protocol-03/
├── onchain/ # 智能合约 (Foundry)
│ ├── src/
│ │ ├── Auction.sol # 主合约 (NFTAuctionMarketUUPS_V1)
│ │ ├── interfaces/ # 接口定义
│ │ └── mocks/ # Mock 合约 (MockUSDC, MockNFT)
│ ├── script/
│ │ └── Deploy.s.sol # 部署脚本
│ ├── test/
│ │ └── NFTAuctionMarketUUPS_V1.t.sol # 单元测试 (13 个)
│ └── foundry.toml
│
├── offchain/ # Go 后端
│ ├── cmd/
│ │ ├── indexer/main.go # Indexer 入口
│ │ └── api/main.go # API 入口
│ ├── internal/
│ │ ├── api/
│ │ │ ├── handler/ # HTTP 处理层 (Gin)
│ │ │ ├── service/ # 业务逻辑层
│ │ │ └── middleware/ # 中间件 (Logger, CORS)
│ │ ├── indexer/
│ │ │ ├── listener.go # 事件监听
│ │ │ └── repository/ # 数据访问层 (Auction, Bid, SyncCursor)
│ │ ├── config/ # 配置管理
│ │ └── contract/ # abigen 生成的合约绑定
│ ├── Dockerfile # Docker 镜像构建
│ └── go.mod
│
├── docker-compose.yml # Docker Compose 编排
├── README.md
└── screenshots/ # 项目截图

text

---

## 🚀 快速开始

### 前置条件

- [Foundry](https://book.getfoundry.sh/getting-started/installation)
- [Go 1.21+](https://golang.org/dl/)
- [MySQL 8.0](https://dev.mysql.com/downloads/)

### 1. 启动本地链 (Anvil)

```bash
cd onchain
anvil
2. 部署合约
bash
forge script script/Deploy.s.sol:DeployAuction \
    --rpc-url http://localhost:8545 \
    --broadcast
3. 启动 Indexer
bash
cd offchain
go run cmd/indexer/main.go
4. 启动 API
bash
go run cmd/api/main.go
5. 创建拍卖
bash
curl -X POST http://localhost:8080/api/auctions \
  -H "Content-Type: application/json" \
  -d '{
    "nft": "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0",
    "token_id": 1,
    "duration": 60,
    "min_bid_usd18": "100000000000000000000",
    "sender": "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
  }'
🧪 测试
运行合约测试
bash
cd onchain
forge test --gas-report -vv
结果：13 个测试全部通过 ✅

https://./screenshots/forge-test1.png

Gas 报告
https://./screenshots/forge-test2.png

函数	平均 Gas
createAuction	161,769
bidEth	53,573
bidUSDC	142,921
endAuction	57,736
initialize	184,645
🌐 部署到 Sepolia
1. 配置环境变量
bash
# onchain/.env
SEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
DEPLOYER_PRIVATE_KEY=YOUR_PRIVATE_KEY
2. 部署
bash
forge script script/Deploy.s.sol:DeployAuction \
    --rpc-url $SEPOLIA_RPC_URL \
    --broadcast \
    --private-key $DEPLOYER_PRIVATE_KEY
3. 部署结果
text
Proxy Address: 0x54B06642502fFd04cf8E2D92b036D50bd36bD78f
USDC Address:  0x67986Fd919ebB385ad023B0Ec774523eF4Bb227c
NFT Address:   0xF983a521AADF4Ba101456D60Ea1876eA2c28F999
https://./screenshots/sepolia-deploy.png

4. Etherscan 验证
https://./screenshots/etherscan-tx.png

https://./screenshots/etherscan-logs.png

📡 API 接口
方法	路径	功能
GET	/health	健康检查
GET	/api/auctions	获取拍卖列表（分页 + 筛选）
GET	/api/auctions/{id}	获取拍卖详情（含出价）
GET	/api/auctions/{id}/bids	获取拍卖的出价历史
POST	/api/auctions	创建拍卖
POST	/api/auctions/{id}/end	结束拍卖
GET	/api/bids	获取所有出价
POST	/api/bids	出价（ETH / USDC）
GET	/api/bids/stats	出价统计
API 响应示例
json
{
  "data": [
    {
      "auction_id": "1",
      "seller": "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
      "nft": "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0",
      "token_id": "1",
      "end_time": "1782195847",
      "min_bid_usd18": "100000000000000000000",
      "highest_bid": "0",
      "ended": false,
      "status": "active",
      "time_remaining": 3600
    }
  ],
  "page": 1,
  "page_size": 10,
  "total": 5
}
🐳 Docker 部署
1. 构建并启动
bash
docker-compose up -d
2. 查看状态
bash
docker-compose ps
3. 查看日志
bash
docker-compose logs -f
4. 停止服务
bash
docker-compose down
📸 截图
API 服务
https://./screenshots/api-log.png

https://./screenshots/api-health.png

https://./screenshots/api-auctions.png

Indexer
https://./screenshots/indexer-startup.png

📄 许可证
MIT License

🙏 致谢
Foundry - 智能合约开发框架

go-ethereum - Go 以太坊客户端

Gin - Go Web 框架

GORM - Go ORM