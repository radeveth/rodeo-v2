// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IPool {
    function exec(address) external view returns (bool);
    function asset() external view returns (address);
    function oracle() external view returns (address);
    function getUpdatedIndex() external view returns (uint256);
    function borrow(uint256) external returns (uint256);
    function repay(uint256) external returns (uint256);
}