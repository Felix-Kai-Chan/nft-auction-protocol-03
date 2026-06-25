// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/**
 * @title MockUSDC
 * @notice 模拟 USDC 代币，支持任意铸造
 */
contract MockUSDC is ERC20 {
    constructor() ERC20("Mock USD Coin", "mUSDC") {}

    /**
     * @dev 任何人都可以铸造代币（仅用于测试/部署脚本）
     */
    function mint(address to, uint256 amount) external {
        _mint(to, amount);
    }
}