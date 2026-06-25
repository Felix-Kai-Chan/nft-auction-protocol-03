#!/bin/bash

# ============================================
# NFT Auction 测试脚本
# 用法: ./test.sh
# ============================================

# ========== 合约地址（当前部署） ==========
NFT=0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0
AUCTION=0xa513E6E4b8f2a923D98304ec87F64353C4D5C853
USDC=0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512

# ========== 账户 ==========
PK1=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
ADDR1=0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
PK2=0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d
ADDR2=0x70997970C51812dc3A010C7d01b50e0d17dc79C8

RPC=http://localhost:8545

# ========== 颜色 ==========
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   NFT Auction 测试脚本${NC}"
echo -e "${BLUE}========================================${NC}"

# ========== 1. 铸造 NFT ==========
echo -e "\n${YELLOW}📦 1. 铸造 NFT (tokenId=30)...${NC}"
cast send $NFT "mint(address,uint256)" $ADDR1 30 --rpc-url $RPC --private-key $PK1

# ========== 2. 授权 ==========
echo -e "\n${YELLOW}🔓 2. 授权 NFT 给拍卖合约...${NC}"
cast send $NFT "approve(address,uint256)" $AUCTION 30 --rpc-url $RPC --private-key $PK1

# ========== 3. 创建拍卖 ==========
echo -e "\n${YELLOW}🔨 3. 创建拍卖 (duration=60s)...${NC}"
cast send $AUCTION "createAuction(address,uint256,uint256,uint256)" $NFT 30 60 100000000000000000000 --rpc-url $RPC --private-key $PK1

# ========== 4. ETH 出价 ==========
echo -e "\n${YELLOW}💰 4. 账户2 出价 1 ETH...${NC}"
cast send $AUCTION "bidEth(uint256)" 1 --value 1ether --rpc-url $RPC --private-key $PK2

# ========== 5. USDC 出价 ==========
echo -e "\n${YELLOW}💰 5. 准备 USDC 出价...${NC}"

echo "  5a. 给账户2 铸造 USDC..."
cast send $USDC "mint(address,uint256)" $ADDR2 10000000000 --rpc-url $RPC --private-key $PK1

echo "  5b. 账户2 授权 USDC 给拍卖合约..."
cast send $USDC "approve(address,uint256)" $AUCTION 10000000000 --rpc-url $RPC --private-key $PK2

echo "  5c. 账户2 USDC 出价 5000 USDC..."
cast send $AUCTION "bidUSDC(uint256,uint256)" 1 5000000000 --rpc-url $RPC --private-key $PK2

# ========== 6. 查询 ==========
echo -e "\n${YELLOW}📊 6. 查询拍卖状态...${NC}"
cast call $AUCTION "auctions(uint256)" 1 --rpc-url $RPC

echo -e "\n${YELLOW}📊 7. 查询 nextAuctionId...${NC}"
cast call $AUCTION "nextAuctionId()" --rpc-url $RPC

# ========== 8. 结束拍卖 ==========
echo -e "\n${YELLOW}⏰ 8. 结束拍卖...${NC}"
echo "  注意: 需要等待 60 秒才能结束"
echo "  cast send $AUCTION \"endAuction(uint256)\" 1 --rpc-url $RPC --private-key $PK1"

# ========== 完成 ==========
echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}✅ 测试完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "\n📌 查看结果: curl http://localhost:8080/api/auctions | jq '.'"