// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "../interfaces/AggregatorInterface.sol";

/**
 * @title MockAggregator
 * @notice 模拟 Chainlink Price Feed，返回固定价格
 */
contract MockAggregator is AggregatorInterface {
    int256 public price; // 存储当前模拟价格
    uint80 private roundId = 1; // 初始轮次设为 1，避免某些前端或合约判断为 0

    constructor(int256 _price) {
        price = _price;
    }

    // 允许外部更新价格，方便测试不同场景
    function setPrice(int256 _newPrice) external {
        price = _newPrice;
    }

    /**
     * @notice 获取最新一轮数据
     * @dev 严格按照 AggregatorInterface 要求的类型返回 (uint256, uint256)
     */
    function latestRoundData()
        external
        view
        override
        returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        return (
            roundId,
            price,
            uint256(block.timestamp - 100), // startedAt: uint256
            uint256(block.timestamp),       // updatedAt: uint256
            roundId                         // answeredInRound
        );
    }

    /**
     * @notice 获取指定轮次的数据
     * @dev 严格按照你的本地接口要求，updatedAt 返回 uint256
     */
    function getRoundData(uint80 _roundId)
        external
        view
        override
        returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        return (
            _roundId,
            price,
            uint256(block.timestamp - 100), // startedAt: uint256
            uint256(block.timestamp),       // updatedAt: uint256
            _roundId                        // answeredInRound
        );
    }

    function decimals() external pure override returns (uint8) {
        return 8; // Chainlink ETH/USD 通常是 8 位小数
    }

    function description() external pure override returns (string memory) {
        return "Mock ETH/USD Aggregator";
    }

    function version() external pure override returns (uint256) {
        return 1;
    }
}