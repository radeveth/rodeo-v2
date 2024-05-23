// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ERC721TokenReceiver {
    function onERC721Received(address, address, uint256, bytes calldata) external pure returns (bytes4);
}