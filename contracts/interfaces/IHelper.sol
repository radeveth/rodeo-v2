// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IHelper {
    function price(address) external view returns (uint256);
    function value(address, uint256) external view returns (uint256);
    function convert(address, address, uint256) external view returns (uint256);
    function swap(
        address,
        address,
        uint256,
        uint256,
        address
    ) external returns (uint256);
}