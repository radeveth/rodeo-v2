// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IInvestor {
    struct Strategy {
        address implementation;
        uint256 cap;
        uint256 status;
    }

    struct Position {
        address owner;
        uint256 start;
        uint256 strategy;
        address token;
        uint256 collateral;
        uint256 borrow;
        uint256 shares;
        uint256 basis;
    }

    function getPool() external view returns (address);
    function getStrategy(uint256 id) external view returns (Strategy memory);
    function getPosition(uint256 id) external view returns (Position memory);
    function life(uint256 id) external view returns (uint256);
    function open(uint256 strategy, address token, uint256 collateral, uint256 borrow) external returns (uint256);
    function edit(uint256 id, int256 borrow, int256 collateral) external;

    function killRepayment(uint256) external returns (uint256);
    
    function partialKillRepayment(uint256 amount) external view returns (uint256);

    function kill(uint256 id) external returns (address, bytes memory);

    function partialKill(uint256 id, uint256 amount) external returns (address, bytes memory);
}