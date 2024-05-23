// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IStrategy {
    function totalShares() external view returns (uint256);
    function mint(uint256 amount) external returns (uint256 shares);
    function burn(uint256 shares) external returns (uint256 amount);
    function kill(uint256 shares, address to) external returns (bytes memory);
    function rate(uint256) external view returns (uint256);
    function exit(address) external;
    function move(address) external;
    function name() external view returns (string memory);
}