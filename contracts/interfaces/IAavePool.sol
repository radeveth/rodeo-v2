// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IAavePool {
    function flashLoanSimple(address receiver, address asset, uint256 amount, bytes calldata params, uint16 referrer)
        external;
}