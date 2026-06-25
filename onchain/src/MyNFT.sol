// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract MyNFT is ERC721, Ownable {
    // 基础元数据链接
    string private _baseTokenURI;
    // 下一个可用的 Token ID
    uint256 public nextTokenId;
    // URI 是否已冻结（一旦冻结，任何人包括 owner 都无法修改）
    bool public isURIFrozen;

    // 自定义错误 (Gas 优化)
    error URIAlreadyFrozen();
    error BatchMintLengthMismatch();

    constructor(string memory baseURI) ERC721("MyNFT", "MNFT") Ownable(msg.sender) {
        _baseTokenURI = baseURI;
    }

    /**
     * @notice 单个铸造
     */
    function mint(address to) external onlyOwner returns (uint256) {
        uint256 tokenId = ++nextTokenId;
        _safeMint(to, tokenId);
        return tokenId;
    }

    /**
     * @notice 批量铸造 (适合空投或批量生成拍卖品)
     */
    function mintBatch(address[] calldata recipients) external onlyOwner returns (uint256[] memory) {
        uint256 length = recipients.length;
        uint256[] memory tokenIds = new uint256[](length);
        
        for (uint256 i = 0; i < length; ) {
            uint256 tokenId = ++nextTokenId;
            _safeMint(recipients[i], tokenId);
            tokenIds[i] = tokenId;
            unchecked { ++i; } // 优化循环 Gas
        }
        return tokenIds;
    }

    /**
     * @notice 修改基础 URI (仅在未冻结时有效)
     */
    function setBaseURI(string memory newBaseURI) external onlyOwner {
        if (isURIFrozen) revert URIAlreadyFrozen();
        _baseTokenURI = newBaseURI;
    }

    /**
     * @notice 永久冻结 URI (增加买家信任，防止项目方篡改)
     */
    function freezeURI() external onlyOwner {
        if (isURIFrozen) revert URIAlreadyFrozen();
        isURIFrozen = true;
    }

    // 重写 OpenZeppelin 的 _baseURI 方法
    function _baseURI() internal view override returns (string memory) {
        return _baseTokenURI;
    }
}