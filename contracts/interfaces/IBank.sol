// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IBank {
    function exec(address) external view returns (bool);
    function transfer(address, address, uint256) external;
}
