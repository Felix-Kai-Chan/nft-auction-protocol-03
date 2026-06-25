// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "forge-std/Test.sol";
import "../src/Auction.sol";
import "../src/interfaces/AggregatorInterface.sol";
import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC721/ERC721.sol";

contract NFTAuctionMarketUUPS_V1Test is Test {
    NFTAuctionMarketUUPS_V1 public auctionMarket;
    MockAggregator public ethUsdFeed;
    MockAggregator public usdcUsdFeed;
    MockUSDC public usdc;
    MockNFT public nft;
    
    address public owner = address(0x1);
    address public bidder1 = address(0x2);
    address public bidder2 = address(0x3);
    address public feeRecipient = address(0x4);
    
    uint256 public constant FEE_BPS = 100; // 1%
    
    function setUp() public {
        // 1. 部署 Mock 合约
        ethUsdFeed = new MockAggregator(2000 * 1e8); // ETH = 2000 USD
        usdcUsdFeed = new MockAggregator(1 * 1e8);   // USDC = 1 USD
        usdc = new MockUSDC();
        nft = new MockNFT();
        
        // 2. 部署实现合约
        NFTAuctionMarketUUPS_V1 implementation = new NFTAuctionMarketUUPS_V1();
        
        // 3. 编码初始化数据
        bytes memory data = abi.encodeWithSelector(
            NFTAuctionMarketUUPS_V1.initialize.selector,
            address(ethUsdFeed),
            address(usdc),
            address(usdcUsdFeed),
            feeRecipient,
            FEE_BPS
        );
        
        // 4. 部署 ERC1967 代理合约
        ERC1967Proxy proxy = new ERC1967Proxy(address(implementation), data);
        auctionMarket = NFTAuctionMarketUUPS_V1(payable(address(proxy)));
        
        // 5. 准备测试数据
        vm.startPrank(owner);
        nft.mint(owner, 1);
        nft.mint(owner, 2);
        nft.setApprovalForAll(address(auctionMarket), true);
        vm.stopPrank();
        
        // 给测试账户转钱
        vm.deal(owner, 100 ether);
        vm.deal(bidder1, 100 ether);
        vm.deal(bidder2, 100 ether);
        
        usdc.mint(bidder1, 50000 * 1e6); // 50,000 USDC
        usdc.mint(bidder2, 50000 * 1e6);
        
        vm.prank(bidder1);
        usdc.approve(address(auctionMarket), type(uint256).max);
        vm.prank(bidder2);
        usdc.approve(address(auctionMarket), type(uint256).max);
    }
    
    /* ============ 初始化测试 ============ */
    function test_Initialize() public {
        assertEq(address(auctionMarket.ethUsdFeed()), address(ethUsdFeed));
        assertEq(address(auctionMarket.usdc()), address(usdc));
        assertEq(auctionMarket.feeRecipient(), feeRecipient);
        assertEq(auctionMarket.feeBps(), FEE_BPS);
        assertEq(auctionMarket.owner(), address(this)); // setUp 中 msg.sender 是 this
    }
    
    /* ============ 创建拍卖测试 ============ */
    function test_CreateAuction() public {
        vm.startPrank(owner);
        uint256 auctionId = auctionMarket.createAuction(
            address(nft),
            1,
            7 days,
            1000 * 1e18 // min bid = 1000 USD
        );
        vm.stopPrank();
        
        assertEq(auctionId, 1);
        
        // 使用你合约里的 getAuction 方法获取结构体
        NFTAuctionMarketUUPS_V1.Auction memory a = auctionMarket.getAuction(auctionId);
        
        assertEq(a.seller, owner);
        assertEq(a.nft, address(nft));
        assertEq(a.tokenId, 1);
        assertEq(a.endTime, block.timestamp + 7 days);
        assertEq(a.ended, false);
        assertEq(a.minBidUsd18, 1000 * 1e18);
        
        // NFT 应该被转移到合约
        assertEq(nft.ownerOf(1), address(auctionMarket));
    }
    
    function test_CreateAuction_Reverts() public {
        vm.startPrank(owner);
        vm.expectRevert(NFTAuctionMarketUUPS_V1.BadDuration.selector);
        auctionMarket.createAuction(address(nft), 1, 0, 1000 * 1e18);
        
        vm.expectRevert(NFTAuctionMarketUUPS_V1.BadMinBidUsd.selector);
        auctionMarket.createAuction(address(nft), 1, 7 days, 0);
        vm.stopPrank();
    }
    
    /* ============ ETH 出价测试 ============ */
    function test_BidEth() public {
        uint256 auctionId = _createAuction();
        
        // bidder1 出价 1 ETH (在 chainid 31337 下等于 2000 USD)
        vm.startPrank(bidder1);
        auctionMarket.bidEth{value: 1 ether}(auctionId);
        vm.stopPrank();
        
        NFTAuctionMarketUUPS_V1.Auction memory a = auctionMarket.getAuction(auctionId);
        
        assertEq(a.highestBidToken, address(0));
        assertEq(a.highestBidAmount, 1 ether);
        assertEq(a.highestBidUsd18, 2000 * 1e18);
        assertEq(a.highestBidder, bidder1);
    }
    
    function test_BidEth_Reverts() public {
        uint256 auctionId = _createAuction();
        
        vm.startPrank(bidder1);
        // 0.1 ETH = 200 USD < 1000 USD
        vm.expectRevert(NFTAuctionMarketUUPS_V1.BidBelowMin.selector);
        auctionMarket.bidEth{value: 0.1 ether}(auctionId);
        
        vm.expectRevert(NFTAuctionMarketUUPS_V1.ZeroBid.selector);
        auctionMarket.bidEth{value: 0}(auctionId);
        vm.stopPrank();
    }
    
    /* ============ USDC 出价测试 ============ */
    function test_BidUSDC() public {
        uint256 auctionId = _createAuction();
        
        vm.startPrank(bidder1);
        auctionMarket.bidUSDC(auctionId, 2000 * 1e6);
        vm.stopPrank();
        
        NFTAuctionMarketUUPS_V1.Auction memory a = auctionMarket.getAuction(auctionId);
        
        assertEq(a.highestBidToken, address(usdc));
        assertEq(a.highestBidAmount, 2000 * 1e6);
        assertEq(a.highestBidUsd18, 2000 * 1e18);
        assertEq(a.highestBidder, bidder1);
    }
    
    /* ============ 多次出价和退款测试 ============ */
    function test_MultipleBids_RefundPreviousBidder() public {
        uint256 auctionId = _createAuction();
        
        vm.startPrank(bidder1);
        auctionMarket.bidEth{value: 1 ether}(auctionId);
        vm.stopPrank();
        
        uint256 bidder1BalanceBefore = bidder1.balance;
        
        // bidder2 出价 2 ETH
        vm.startPrank(bidder2);
        auctionMarket.bidEth{value: 2 ether}(auctionId);
        vm.stopPrank();
        
        // bidder1 应该收到退款
        assertEq(bidder1.balance, bidder1BalanceBefore + 1 ether);
        
        NFTAuctionMarketUUPS_V1.Auction memory a = auctionMarket.getAuction(auctionId);
        assertEq(a.highestBidder, bidder2);
    }
    
    /* ============ 拍卖结算测试 ============ */
    function test_EndAuction_WinnerGetsNFT_ETH() public {
        uint256 auctionId = _createAuction();
        
        vm.startPrank(bidder1);
        auctionMarket.bidEth{value: 1 ether}(auctionId);
        vm.stopPrank();
        
        // 快进到拍卖结束
        vm.warp(block.timestamp + 7 days + 1);
        
        uint256 sellerBalanceBefore = owner.balance;
        uint256 feeRecipientBalanceBefore = feeRecipient.balance;
        
        // 任何人都可以调用 endAuction
        auctionMarket.endAuction(auctionId);
        
        // 检查 NFT 归 winner
        assertEq(nft.ownerOf(1), bidder1);
        
        // 检查资金分配 (1% 手续费)
        uint256 expectedFee = (1 ether * FEE_BPS) / 10000;
        uint256 expectedSellerAmount = 1 ether - expectedFee;
        assertEq(owner.balance, sellerBalanceBefore + expectedSellerAmount);
        assertEq(feeRecipient.balance, feeRecipientBalanceBefore + expectedFee);
    }
    
    function test_EndAuction_NoBids_Cancelled() public {
        uint256 auctionId = _createAuction();
        
        vm.warp(block.timestamp + 7 days + 1);
        auctionMarket.endAuction(auctionId);
        
        // 没人出价，NFT 退回卖家
        assertEq(nft.ownerOf(1), owner);
        
        NFTAuctionMarketUUPS_V1.Auction memory a = auctionMarket.getAuction(auctionId);
        assertTrue(a.ended);
    }
    
    function test_EndAuction_Reverts() public {
        uint256 auctionId = _createAuction();
        
        // 还没结束
        vm.expectRevert(NFTAuctionMarketUUPS_V1.AuctionNotEndedYet.selector);
        auctionMarket.endAuction(auctionId);
        
        // 结束一次
        vm.warp(block.timestamp + 7 days + 1);
        auctionMarket.endAuction(auctionId);
        
        // 再次结束
        vm.expectRevert(NFTAuctionMarketUUPS_V1.AuctionAlreadyEnded.selector);
        auctionMarket.endAuction(auctionId);
    }
    
    /* ============ 辅助函数 ============ */
    function _createAuction() internal returns (uint256) {
        vm.startPrank(owner);
        uint256 auctionId = auctionMarket.createAuction(
            address(nft),
            1,
            7 days,
            1000 * 1e18
        );
        vm.stopPrank();
        return auctionId;
    }
}

/* ============ Mock 合约 ============ */

contract MockAggregator is AggregatorInterface {
    int256 private price;
    uint80 private roundId = 0;
    
    constructor(int256 _price) { price = _price; }
    
    function latestRoundData() external view returns (uint80, int256, uint256, uint256, uint80) {
        return (roundId, price, block.timestamp - 100, block.timestamp, roundId);
    }
    
    function decimals() external pure returns (uint8) { return 8; }
    function description() external pure returns (string memory) { return "Mock Aggregator"; }
    function version() external pure returns (uint256) { return 1; }
    
    function getRoundData(uint80 _roundId) external view returns (uint80, int256, uint256, uint256, uint80) {
        return (_roundId, price, block.timestamp - 100, block.timestamp, _roundId);
    }
}

contract MockUSDC is ERC20 {
    constructor() ERC20("Mock USDC", "USDC") {}
    function mint(address to, uint256 amount) public { _mint(to, amount); }
}

contract MockNFT is ERC721 {
    constructor() ERC721("Mock NFT", "MNFT") {}
    function mint(address to, uint256 tokenId) public { _mint(to, tokenId); }
}