// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";

/**
 * @title MockNFT
 * @notice 简单的 NFT 合约，支持公开铸造
 */
contract MockNFT is ERC721 {
    constructor() ERC721("Mock NFT Collection", "MNFT") {}

    /**
     * @dev 任何人都可以铸造 NFT（仅用于测试/部署脚本）
     */
    function mint(address to, uint256 tokenId) external {
        _mint(to, tokenId);
    }
}