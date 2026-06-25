// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol"; // 新增：安全ERC20库

import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/utils/ReentrancyGuardUpgradeable.sol";

import "./interfaces/AggregatorInterface.sol";

contract NFTAuctionMarketUUPS_V1 is
    Initializable,
    OwnableUpgradeable,
    UUPSUpgradeable,
    ReentrancyGuardUpgradeable
{
    using SafeERC20 for IERC20; // 绑定安全库

    // address(0) 表示 ETH
    address public constant NATIVE_TOKEN = address(0);

    // 1. 自定义错误 (Gas 优化)
    error BadDuration();
    error BadMinBidUsd();
    error AuctionDoesNotExist();
    error AuctionAlreadyEnded();
    error AuctionNotEndedYet();
    error ZeroBid();
    error BidBelowMin();
    error BidNotHighEnough();
    error TransferFailed();

    struct Auction {
        address seller;
        address nft;
        uint256 tokenId;
        uint256 endTime;
        bool ended;
        address highestBidToken;
        uint256 highestBidAmount;
        uint256 highestBidUsd18;
        address highestBidder;
        uint256 minBidUsd18;
    }

    uint256 public nextAuctionId;
    mapping(uint256 => Auction) public auctions;

    IERC20 public usdc;
    AggregatorInterface public ethUsdFeed;
    AggregatorInterface public usdcUsdFeed;
    address public feeRecipient;
    uint256 public feeBps; 

    event AuctionCreated(uint256 indexed auctionId, address indexed seller, address indexed nft, uint256 tokenId, uint256 endTime, uint256 minBidUsd18);
    event BidPlaced(uint256 indexed auctionId, address indexed bidder, address bidToken, uint256 bidAmount, uint256 bidUsd18);
    event AuctionEnded(uint256 indexed auctionId, address winner, address payToken, uint256 payAmount, uint256 payUsd18);
    event AuctionCancelled(uint256 indexed auctionId);

    /// @custom:oz-upgrades-unsafe-allow constructor
    constructor() {
        _disableInitializers();
    }

    function initialize(
        address _ethUsdFeed,
        address _usdc,
        address _usdcUsdFeed,
        address _feeRecipient,
        uint256 _feeBps
    ) public initializer {
        __Ownable_init(msg.sender);
        __UUPSUpgradeable_init();
        __ReentrancyGuard_init();

        ethUsdFeed = AggregatorInterface(_ethUsdFeed);
        usdc = IERC20(_usdc);
        usdcUsdFeed = AggregatorInterface(_usdcUsdFeed);
        feeRecipient = _feeRecipient;
        feeBps = _feeBps;
    }

    function _authorizeUpgrade(address newImplementation) internal override onlyOwner {}

    /* ---------------- Oracle helpers (USD 18 decimals) ---------------- */
    function _scaleTo18(uint256 value, uint8 decimals_) internal pure returns (uint256) {
        if (decimals_ == 18) return value;
        if (decimals_ < 18) return value * (10 ** (18 - decimals_));
        return value / (10 ** (decimals_ - 18));
    }

    function _readFeed(AggregatorInterface feed) internal view returns (uint256 answer, uint8 dec) {
        (, int256 a,, uint256 updatedAt,) = feed.latestRoundData();
        require(a > 0, "Oracle: bad answer");
        require(updatedAt > 0, "Oracle: stale");
        answer = uint256(a);
        dec = feed.decimals();
    }

    function _ethToUsd18(uint256 weiAmount) internal view returns (uint256) {
        if (block.chainid == 31337) return weiAmount * 2000;
        (uint256 price, uint8 dec) = _readFeed(ethUsdFeed);
        uint256 usd = (weiAmount * price) / 1e18;
        return _scaleTo18(usd, dec);
    }

    function _usdcToUsd18(uint256 usdcAmount) internal view returns (uint256) {
        if (block.chainid == 31337) return usdcAmount * 1e12;
        (uint256 price, uint8 feedDec) = _readFeed(usdcUsdFeed);
        uint256 usdWithFeedDec = (usdcAmount * price) / 1e6;
        return _scaleTo18(usdWithFeedDec, feedDec);
    }

    /* ---------------- Market core ---------------- */
    function createAuction(
        address nft,
        uint256 tokenId,
        uint256 durationSeconds,
        uint256 minBidUsd18
    ) external nonReentrant returns (uint256 auctionId) {
        if (durationSeconds == 0) revert BadDuration();
        if (minBidUsd18 == 0) revert BadMinBidUsd();

        auctionId = ++nextAuctionId;
        auctions[auctionId] = Auction({
            seller: msg.sender, nft: nft, tokenId: tokenId,
            endTime: block.timestamp + durationSeconds, ended: false,
            highestBidToken: NATIVE_TOKEN, highestBidAmount: 0,
            highestBidUsd18: 0, highestBidder: address(0), minBidUsd18: minBidUsd18
        });

        // 2. 使用 safeTransferFrom 防止恶意 NFT
        IERC721(nft).safeTransferFrom(msg.sender, address(this), tokenId);
        emit AuctionCreated(auctionId, msg.sender, nft, tokenId, block.timestamp + durationSeconds, minBidUsd18);
    }

    function bidEth(uint256 auctionId) external payable nonReentrant {
        Auction storage a = auctions[auctionId];
        if (a.seller == address(0)) revert AuctionDoesNotExist();
        if (a.ended) revert AuctionAlreadyEnded();
        if (block.timestamp >= a.endTime) revert AuctionNotEndedYet();
        if (msg.value == 0) revert ZeroBid();

        uint256 bidUsd18 = _ethToUsd18(msg.value);
        _placeBid(a, auctionId, msg.sender, NATIVE_TOKEN, msg.value, bidUsd18);
    }

    function bidUSDC(uint256 auctionId, uint256 usdcAmount) external nonReentrant {
        Auction storage a = auctions[auctionId];
        if (a.seller == address(0)) revert AuctionDoesNotExist();
        if (a.ended) revert AuctionAlreadyEnded();
        if (block.timestamp >= a.endTime) revert AuctionNotEndedYet();
        if (usdcAmount == 0) revert ZeroBid();

        uint256 bidUsd18 = _usdcToUsd18(usdcAmount);
        // 3. 使用 SafeERC20 划扣资金
        IERC20(usdc).safeTransferFrom(msg.sender, address(this), usdcAmount);
        _placeBid(a, auctionId, msg.sender, address(usdc), usdcAmount, bidUsd18);
    }

    function _placeBid(Auction storage a, uint256 auctionId, address bidder, address bidToken, uint256 bidAmount, uint256 bidUsd18) internal {
        if (bidUsd18 < a.minBidUsd18) revert BidBelowMin();
        if (bidUsd18 <= a.highestBidUsd18) revert BidNotHighEnough();

        if (a.highestBidder != address(0)) {
            _refund(a.highestBidder, a.highestBidToken, a.highestBidAmount);
        }

        a.highestBidder = bidder;
        a.highestBidToken = bidToken;
        a.highestBidAmount = bidAmount;
        a.highestBidUsd18 = bidUsd18;
        emit BidPlaced(auctionId, bidder, bidToken, bidAmount, bidUsd18);
    }

    function _refund(address to, address token, uint256 amount) internal {
        if (amount == 0) return;
        if (token == NATIVE_TOKEN) {
            (bool ok,) = payable(to).call{value: amount}("");
            if (!ok) revert TransferFailed();
        } else {
            // 3. 使用 SafeERC20 退款
            IERC20(token).safeTransfer(to, amount);
        }
    }

    function endAuction(uint256 auctionId) external nonReentrant {
        Auction storage a = auctions[auctionId];
        if (a.seller == address(0)) revert AuctionDoesNotExist();
        if (a.ended) revert AuctionAlreadyEnded();
        if (block.timestamp < a.endTime) revert AuctionNotEndedYet();

        a.ended = true;

        if (a.highestBidder == address(0)) {
            IERC721(a.nft).safeTransferFrom(address(this), a.seller, a.tokenId);
            emit AuctionCancelled(auctionId);
            return;
        }

        // NFT 给赢家
        IERC721(a.nft).safeTransferFrom(address(this), a.highestBidder, a.tokenId);

        // 4. 结算资金并扣除手续费
        uint256 amount = a.highestBidAmount;
        if (a.highestBidToken == NATIVE_TOKEN) {
            uint256 fee = (amount * feeBps) / 10000;
            if (fee > 0) {
                (bool okFee,) = payable(feeRecipient).call{value: fee}("");
                if (!okFee) revert TransferFailed();
            }
            (bool okSeller,) = payable(a.seller).call{value: amount - fee}("");
            if (!okSeller) revert TransferFailed();
        } else {
            uint256 fee = (amount * feeBps) / 10000;
            if (fee > 0) {
                IERC20(a.highestBidToken).safeTransfer(feeRecipient, fee);
            }
            IERC20(a.highestBidToken).safeTransfer(a.seller, amount - fee);
        }

        emit AuctionEnded(auctionId, a.highestBidder, a.highestBidToken, amount, a.highestBidUsd18);
    }

    function getAuction(uint256 auctionId) external view returns (Auction memory) {
        return auctions[auctionId];
    }

    // ✅ 新增：接收 ERC721 NFT 的回调函数
    function onERC721Received(
        address,
        address,
        uint256,
        bytes memory
    ) external pure returns (bytes4) {
        return this.onERC721Received.selector;
    }

    receive() external payable {}
}