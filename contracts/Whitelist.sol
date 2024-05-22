// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title Whitelist
 * @notice This contract manages a whitelist and authorized executors
 */
contract Whitelist {
    /// @notice Mapping to manage execution permissions
    mapping(address => bool) public exec;

    /// @notice Mapping to manage whitelist status
    mapping(address => bool) public whitelist;

    /// @notice Event emitted when a configuration is changed
    /// @param what The configuration key
    /// @param data The address value set
    event File(bytes32 indexed what, address data);

    /// @notice Event emitted when a batch of addresses is whitelisted or removed from the whitelist
    /// @param addresses The array of addresses
    /// @param status The new whitelist status
    event BatchWhitelist(address[] addresses, bool status);

    /// @notice Error for invalid configuration key
    error InvalidFile();

    /// @notice Error for unauthorized access
    error Unauthorized();

    /**
     * @notice Constructor to initialize the contract
     * Grants the deployer the execution permission
     */
    constructor() {
        exec[msg.sender] = true;
    }

    /**
     * @notice Modifier to check if the caller is authorized
     */
    modifier auth() {
        if (!exec[msg.sender]) revert Unauthorized();
        _;
    }

    /**
     * @notice Function to update execution or whitelist status
     * @param what The configuration key ("exec" or "whitelist")
     * @param data The address value to set
     */
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

    /**
     * @notice Function to batch update the whitelist status
     * @param addresses The array of addresses to update
     * @param status The new whitelist status (true to whitelist, false to remove)
     */
    function batchWhitelist(address[] memory addresses, bool status) external auth {
        uint256 l = addresses.length;
        for (uint256 i = 0; i < l; i++) {
            whitelist[addresses[i]] = status;
        }
        emit BatchWhitelist(addresses, status);
    }

    /**
     * @notice Function to check if an address is whitelisted
     * @param user The address to check
     * @return True if the address is whitelisted, false otherwise
     */
    function check(address user) external view returns (bool) {
        return whitelist[user];
    }
}
