// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Whitelist {
    mapping(address => bool) public exec;
    mapping(address => bool) public whitelist;

    event File(bytes32 indexed what, address data);
    event BatchWhitelist(address[] addresses, bool status);

    error InvalidFile();
    error Unauthorized();

    constructor() {
        exec[msg.sender] = true;
    }

    modifier auth() {
        if (!exec[msg.sender]) revert Unauthorized();
        _;
    }

    function file(bytes32 what, address data) external auth {
        if (what == "exec") {
            exec[data] = !exec[data];
        } else if (what == "whitelist") {
            whitelist[data] = !whitelist[data];
        } else {
            revert InvalidFile();
        }
        emit File(what, data);
    }

    function batchWhitelist(address[] memory addresses, bool status) external auth {
        uint256 l = addresses.length;
        for (uint256 i = 0; i < l; i++) {
            whitelist[addresses[i]] = status;
        }
        emit BatchWhitelist(addresses, status);
    }

    function check(address user) external view returns (bool) {
        return whitelist[user];
    }
}
