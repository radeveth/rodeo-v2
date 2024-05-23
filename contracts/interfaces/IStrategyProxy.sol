// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IStrategyProxy {
    function exec(address) external view returns (bool);
    function mint(address, uint256) external returns (uint256);
    function burn(address, uint256) external returns (uint256);
    function kill(address, uint256, address) external returns (bytes memory);
}