// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "forge-std/Script.sol";
import "../src/Auction.sol";
import "../src/interfaces/AggregatorInterface.sol";
import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC721/ERC721.sol";

// ============ Mock 合约定义 ============

contract MockAggregator is AggregatorInterface {
    int256 public price;
    constructor(int256 _price) { price = _price; }
    function latestRoundData() external view returns (uint80, int256, uint256, uint256, uint80) {
        return (0, price, block.timestamp, block.timestamp, 0);
    }
    function decimals() external pure returns (uint8) { return 8; }
    function description() external pure returns (string memory) { return "Mock Feed"; }
    function version() external pure returns (uint256) { return 1; }
    function getRoundData(uint80) external view returns (uint80, int256, uint256, uint256, uint80) {
        return (0, price, block.timestamp, block.timestamp, 0);
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

// ============ 部署脚本主体 ============
contract DeployAuction is Script {
    function run() external {
        // ✅ 从环境变量读取私钥（不要硬编码）
        uint256 deployerPrivateKey = vm.envUint("DEPLOYER_PRIVATE_KEY");
        vm.startBroadcast();

        // 2. 部署 Mock 依赖
        MockAggregator ethFeed = new MockAggregator(2000 * 1e8);
        MockUSDC usdc = new MockUSDC();
        MockNFT nft = new MockNFT();

        // 3. 铸币
        address deployer = vm.addr(deployerPrivateKey);
        usdc.mint(deployer, 10000 * 1e6);
        nft.mint(deployer, 1);
        nft.mint(deployer, 2);

        // 4. 部署实现合约
        NFTAuctionMarketUUPS_V1 implementation = new NFTAuctionMarketUUPS_V1();

        // 5. 编码初始化数据
        bytes memory data = abi.encodeWithSelector(
            NFTAuctionMarketUUPS_V1.initialize.selector,
            address(ethFeed),
            address(usdc),
            address(ethFeed),
            deployer,
            100
        );

        // 6. 部署代理合约
        ERC1967Proxy proxy = new ERC1967Proxy(address(implementation), data);

        vm.stopBroadcast();

        console.log("==================================");
        console.log(unicode"🎉 部署成功!");
        console.log("Proxy Address:", address(proxy));
        console.log("USDC Address: ", address(usdc));
        console.log("NFT Address:  ", address(nft));
        console.log("==================================");
    }
}