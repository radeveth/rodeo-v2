// SPDX-License-Identifier: MIT
pragma solidity 0.8.17;

/**
 * @title InvestorStrategyProxy
 * @dev Proxy contract for interacting with investment strategies.
 */
interface IERC20 {
    /**
     * @dev Approves the passed address to spend the specified amount of tokens on behalf of the caller.
     * @param spender The address which will spend the funds.
     * @param amount The amount of tokens to be spent.
     * @return A boolean that indicates if the operation was successful.
     */
    function approve(address spender, uint256 amount) external returns (bool);

    /**
     * @dev Transfers tokens to a specified address.
     * @param to The address to transfer to.
     * @param amount The amount to be transferred.
     * @return A boolean that indicates if the operation was successful.
     */
    function transfer(address to, uint256 amount) external returns (bool);
}

/**
 * @title IStrategy
 * @dev Interface for investment strategy contracts.
 */
interface IStrategy {
    /**
     * @dev Mints shares in the strategy in exchange for the specified amount of assets.
     * @param amount The amount of assets to be invested.
     * @return The number of shares minted.
     */
    function mint(uint256 amount) external returns (uint256 shares);

    /**
     * @dev Burns the specified number of shares in the strategy in exchange for assets.
     * @param shares The number of shares to be burned.
     * @return The amount of assets returned.
     */
    function burn(uint256 shares) external returns (uint256 amount);

    /**
     * @dev Kills the specified number of shares and transfers assets to the target address.
     * @param shares The number of shares to be killed.
     * @param to The address to transfer the assets to.
     * @return A bytes array containing any additional data returned by the strategy.
     */
    function kill(uint256 shares, address to) external returns (bytes memory);
}

/**
 * @title InvestorStrategyProxy
 * @dev Proxy contract for managing interactions with investment strategies.
 */
contract InvestorStrategyProxy {
    /// @notice The asset managed by the proxy.
    IERC20 public asset;

    /// @notice Mapping of authorized executors.
    mapping(address => bool) public exec;

    /**
     * @dev Emitted when a configuration change is made.
     * @param what The identifier of the configuration change.
     * @param data The address associated with the change.
     */
    event File(bytes32 indexed what, address data);

    /**
     * @dev Error thrown when an invalid file operation is attempted.
     */
    error InvalidFile();

    /**
     * @dev Error thrown when an unauthorized operation is attempted.
     */
    error Unauthorized();

    /**
     * @dev Sets the asset and grants the deployer executor rights.
     * @param _asset The address of the asset managed by the proxy.
     */
    constructor(address _asset) {
        asset = IERC20(_asset);
        exec[msg.sender] = true;
    }

    /**
     * @dev Modifier to restrict access to authorized executors.
     */
    modifier auth() {
        if (!exec[msg.sender]) revert Unauthorized();
        _;
    }

    /**
     * @dev Updates the executor status of an address.
     * @param what The configuration to be updated ("exec").
     * @param data The address to be added or removed as an executor.
     */
    function file(bytes32 what, address data) public auth {
        if (what == "exec") {
            exec[data] = !exec[data];
        } else {
            revert InvalidFile();
        }
        emit File(what, data);
    }

    /**
     * @dev Mints shares in the specified strategy using the provided amount of assets.
     * @param strategy The address of the strategy contract.
     * @param amount The amount of assets to invest.
     * @return The number of shares minted.
     */
    function mint(address strategy, uint256 amount) public auth returns (uint256) {
        asset.approve(strategy, amount);
        return IStrategy(strategy).mint(amount);
    }

    /**
     * @dev Burns the specified number of shares in the strategy and transfers the returned assets to the caller.
     * @param strategy The address of the strategy contract.
     * @param shares The number of shares to be burned.
     * @return The amount of assets returned.
     */
    function burn(address strategy, uint256 shares) public auth returns (uint256) {
        uint256 amount = IStrategy(strategy).burn(shares);
        asset.transfer(msg.sender, amount);
        return amount;
    }

    /**
     * @dev Kills the specified number of shares in the strategy and transfers the assets to the target address.
     * @param strategy The address of the strategy contract.
     * @param shares The number of shares to be killed.
     * @param target The address to transfer the assets to.
     * @return A bytes array containing any additional data returned by the strategy.
     */
    function kill(address strategy, uint256 shares, address target) public auth returns (bytes memory) {
        return IStrategy(strategy).kill(shares, target);
    }

    /**
     * @dev Executes a call to the specified strategy contract with the provided data and value.
     * @param strategy The address of the strategy contract.
     * @param value The amount of ETH to send with the call.
     * @param data The data to send with the call.
     */
    function call(address strategy, uint256 value, bytes calldata data) public auth {
        (bool success,) = strategy.call{value: value}(data);
        if (!success) {
            assembly {
                returndatacopy(0, 0, returndatasize())
                revert(0, returndatasize())
            }
        }
    }
}
