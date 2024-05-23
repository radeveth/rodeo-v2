// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IStore {
    function exec(address) external view returns (bool);
    function getUint(bytes32) external view returns (uint256);
    function getAddress(bytes32) external view returns (address);
    function setUint(bytes32, uint256) external returns (uint256);
    function setUintDelta(bytes32, int256) external returns (uint256);
    function setAddress(bytes32, address) external returns (address);
}