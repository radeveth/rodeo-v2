// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title Bank
 * @notice This contract manages fund transfers and execution permissions.
 */
contract Bank {
    /// @notice Mapping to manage execution permissions
    mapping(address => bool) public exec;

    /// @notice Event emitted when a configuration is changed
    /// @param what The configuration key
    /// @param data The address value set
    event File(bytes32 indexed what, address data);

    /// @notice Error for invalid configuration key
    error InvalidFile();

    /// @notice Error for unauthorized access
    error Unauthorized();

    /// @notice Error for failed transfer
    error TransferFailed();

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
     * @notice Function to update execution permissions
     * @param what The configuration key ("exec")
     * @param data The address value to set
     */
    function file(bytes32 what, address data) external auth {
        if (what == "exec") {
            exec[data] = !exec[data];
        } else {
            revert InvalidFile();
        }
        emit File(what, data);
    }

    /**
     * @notice Function to transfer native currency
     * @param to The address to transfer to
     * @param amount The amount to transfer
     */
    function transferNative(address to, uint256 amount) external auth {
        if (amount == 0) return;
        (bool s,) = to.call{value: amount}("");
        if (!s) revert TransferFailed();
    }

    /**
     * @notice Function to transfer ERC20 tokens
     * @param token The address of the token contract
     * @param to The address to transfer to
     * @param amount The amount to transfer
     */
    function transfer(address token, address to, uint256 amount) external auth {
        if (amount == 0) return;
        if (!IERC20(token).transfer(to, amount)) revert TransferFailed();
    }
}

/**
 * @title IERC20
 * @notice Interface for ERC20 token contract to perform token transfers
 */
interface IERC20 {
    /**
     * @notice Transfer tokens to a specified address
     * @param to The address to transfer to
     * @param amount The amount to be transferred
     * @return True if the transfer was successful, false otherwise
     */
    function transfer(address to, uint256 amount) external returns (bool);
}
