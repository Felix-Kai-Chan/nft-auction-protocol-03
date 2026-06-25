// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "forge-std/Test.sol";
import "../src/MyNFT.sol";

contract MyNFTTest is Test {
    MyNFT public nft;
    address public owner = address(0x1);
    address public user = address(0x2);

    function setUp() public {
        vm.prank(owner);
        nft = new MyNFT("ipfs://QmTest/");
    }

    // 测试基础铸造
    function test_Mint() public {
        vm.prank(owner);
        uint256 id = nft.mint(user);
        
        assertEq(id, 1);
        assertEq(nft.ownerOf(1), user);
        assertEq(nft.tokenURI(1), "ipfs://QmTest/1");
    }

    // 测试批量铸造
    function test_MintBatch() public {
        address[] memory recipients = new address[](3);
        recipients[0] = user;
        recipients[1] = address(0x3);
        recipients[2] = address(0x4);

        vm.prank(owner);
        uint256[] memory ids = nft.mintBatch(recipients);

        assertEq(ids.length, 3);
        assertEq(nft.ownerOf(1), user);
        assertEq(nft.ownerOf(3), address(0x4));
    }

    // 测试 URI 冻结机制
    function test_FreezeURI() public {
        vm.startPrank(owner);
        nft.freezeURI();
        
        // 冻结后尝试修改，应该 revert
        vm.expectRevert(MyNFT.URIAlreadyFrozen.selector);
        nft.setBaseURI("ipfs://NewURI/");
        vm.stopPrank();
    }
}