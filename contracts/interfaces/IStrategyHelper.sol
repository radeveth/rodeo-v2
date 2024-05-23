// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IStrategyHelper {
    function swap(address, address, uint256, uint256, address) external returns (uint256);
}