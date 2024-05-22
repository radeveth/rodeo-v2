// SPDX-License-Identifier: MIT
pragma solidity 0.8.17;

/**
 * @title StrategySushiswap
 * @notice This contract interacts with the SushiSwap protocol to manage investments,
 * handle deposits, withdrawals, and perform various strategy functions.
 */
contract StrategySushiswap {
    /// @notice Name of the strategy
    string public name;

    /// @notice Total number of shares
    uint256 public totalShares = 1_000_000;

    /// @notice Slippage tolerance percentage (in basis points)
    uint256 public slippage = 500;

    /// @dev Reentrancy guard
    bool internal entered;

    /// @notice ERC20 asset token
    IERC20 public asset;

    /// @notice Strategy helper contract
    IStrategyHelper public strategyHelper;

    /// @notice SushiSwap rewarder contract
    ISushiswapMiniChefV2 public rewarder;

    /// @notice Uniswap V2 pair (liquidity pool) contract
    IUniswapV2Pair public pool;

    /// @notice ID of the SushiSwap pool
    uint256 public poolId;

    /// @notice Mapping to manage execution permissions
    mapping(address => bool) public exec;

    /// @notice Mapping to manage keeper permissions
    mapping(address => bool) public keepers;

    /// @notice Event emitted when a configuration is changed with a uint256 data
    event File(bytes32 indexed what, uint256 data);

    /// @notice Event emitted when a configuration is changed with an address data
    event File(bytes32 indexed what, address data);

    /// @notice Event emitted when shares are minted
    event Mint(uint256 amount, uint256 shares);

    /// @notice Event emitted when shares are burned
    event Burn(uint256 amount, uint256 shares);

    /// @notice Event emitted when shares are killed
    event Kill(uint256 amount, uint256 shares);

    /// @notice Event emitted when profit is earned
    event Earn(uint256 tvl, uint256 profit);

    /// @notice Error for unauthorized keeper
    error NotKeeper();

    /// @notice Error for invalid file parameter
    error InvalidFile();

    /// @notice Error for reentrancy
    error NoReentering();

    /// @notice Error for unauthorized access
    error Unauthorized();

    /**
     * @notice Constructor to initialize the strategy contract
     * @param _asset Address of the asset token
     * @param _strategyHelper Address of the strategy helper contract
     * @param _rewarder Address of the SushiSwap rewarder contract
     * @param _poolId ID of the SushiSwap pool
     */
    constructor(address _asset, address _strategyHelper, address _rewarder, uint256 _poolId) {
        exec[msg.sender] = true;
        asset = IERC20(_asset);
        strategyHelper = IStrategyHelper(_strategyHelper);
        rewarder = ISushiswapMiniChefV2(_rewarder);
        poolId = _poolId;
        pool = IUniswapV2Pair(rewarder.lpToken(poolId));
        name =
            string(abi.encodePacked("SushiSwap ", IERC20(pool.token0()).symbol(), "/", IERC20(pool.token1()).symbol()));
    }

    /**
     * @notice Modifier to prevent reentrancy
     */
    modifier loop() {
        if (entered) revert NoReentering();
        entered = true;
        _;
        entered = false;
    }

    /**
     * @notice Modifier to check if the caller is authorized
     */
    modifier auth() {
        if (!exec[msg.sender]) revert Unauthorized();
        _;
    }

    /**
     * @notice Function to update contract configurations with address parameters
     * @param what The configuration key
     * @param data The address value to set
     */
    function file(bytes32 what, address data) external auth {
        if (what == "exec") {
            exec[data] = !exec[data];
        } else if (what == "keeper") {
            keepers[data] = !keepers[data];
        } else {
            revert InvalidFile();
        }
        emit File(what, data);
    }

    /**
     * @notice Function to update contract configurations with uint256 parameters
     * @param what The configuration key
     * @param data The uint256 value to set
     */
    function file(bytes32 what, uint256 data) external auth {
        if (what == "slippage") {
            slippage = data;
        } else {
            revert InvalidFile();
        }
        emit File(what, data);
    }

    /**
     * @notice Function to calculate the rate of shares
     * @param sha The number of shares
     * @return The rate of the shares
     */
    function rate(uint256 sha) public view returns (uint256) {
        if (sha == 0 || totalShares == 0) return 0;
        IUniswapV2Pair pair = pool;
        uint256 tot = pair.totalSupply();
        uint256 amt = totalManagedAssets();
        uint256 reserve0;
        uint256 reserve1;
        {
            (uint112 r0, uint112 r1,) = pair.getReserves();
            reserve0 = uint256(r0) * 1e18 / (10 ** IERC20(pair.token0()).decimals());
            reserve1 = uint256(r1) * 1e18 / (10 ** IERC20(pair.token1()).decimals());
        }
        uint256 price0 = strategyHelper.price(pair.token0());
        uint256 price1 = strategyHelper.price(pair.token1());
        uint256 val = 2 * ((sqrt(reserve0 * reserve1) * sqrt(price0 * price1)) / tot);
        return sha * (val * amt / 1e18) / totalShares;
    }

    /**
     * @notice Function to mint new shares
     * @param amount The amount of asset to mint shares for
     * @return The number of shares minted
     */
    function mint(uint256 amount) public auth loop returns (uint256) {
        asset.transferFrom(msg.sender, address(this), amount);
        IUniswapV2Pair pair = pool;
        IERC20 tok0 = IERC20(pair.token0());
        IERC20 tok1 = IERC20(pair.token1());
        uint256 slp = slippage;
        uint256 tma = totalManagedAssets();
        {
            uint256 haf = amount / 2;
            asset.approve(address(strategyHelper), amount);
            strategyHelper.swap(address(asset), address(tok0), haf, slp, address(this));
            strategyHelper.swap(address(asset), address(tok1), amount - haf, slp, address(this));
            IERC20(tok0).transfer(address(pair), tok0.balanceOf(address(this)));
            IERC20(tok1).transfer(address(pair), tok1.balanceOf(address(this)));
        }
        pair.mint(address(this));
        pair.skim(address(this));
        uint256 liq = IERC20(address(pair)).balanceOf(address(this));
        IERC20(address(pair)).approve(address(rewarder), liq);
        rewarder.deposit(poolId, liq, address(this));
        uint256 shares = tma == 0 ? liq : liq * totalShares / tma;

        totalShares += shares;
        emit Mint(amount, shares);
        return shares;
    }

    /**
     * @notice Function to burn shares
     * @param shares The number of shares to burn
     * @return The amount of asset received from burning the shares
     */
    function burn(uint256 shares) public auth loop returns (uint256) {
        IUniswapV2Pair pair = pool;
        uint256 slp = slippage;
        {
            uint256 tma = totalManagedAssets();
            uint256 amt = shares * tma / totalShares;
            rewarder.withdraw(poolId, amt, address(pair));
            pair.burn(address(this));
        }
        IERC20 tok0 = IERC20(pair.token0());
        IERC20 tok1 = IERC20(pair.token1());
        uint256 bal0 = tok0.balanceOf(address(this));
        uint256 bal1 = tok1.balanceOf(address(this));
        tok0.approve(address(strategyHelper), bal0);
        tok1.approve(address(strategyHelper), bal1);
        uint256 amt0 = strategyHelper.swap(address(tok0), address(asset), bal0, slp, msg.sender);
        uint256 amt1 = strategyHelper.swap(address(tok1), address(asset), bal1, slp, msg.sender);
        uint256 amount = amt0 + amt1;

        totalShares -= shares;
        emit Burn(amount, shares);
        return amount;
    }

    /**
     * @notice Function to kill shares
     * @param shares The number of shares to kill
     * @param to The address to send the assets to
     * @return The data from the kill action
     */
    function kill(uint256 shares, address to) external auth loop returns (bytes memory) {
        uint256 amount = shares * totalManagedAssets() / totalShares;
        rewarder.withdraw(poolId, amount, to);

        totalShares -= shares;
        emit Kill(amount, shares);

        address[] memory assets = new address[](1);
        assets[0] = address(pool);
        return abi.encode(bytes32("sushi"), assets);
    }

    /**
     * @notice Function to perform earning actions
     */
    function earn() public payable loop {
        if (!keepers[msg.sender]) revert NotKeeper();
        uint256 before = rate(totalShares);

        IUniswapV2Pair pair = pool;
        IERC20 rew = IERC20(rewarder.SUSHI());
        rewarder.harvest(poolId, address(this));
        uint256 amt = rew.balanceOf(address(this));
        uint256 haf = amt / 2;
        if (strategyHelper.value(address(rew), amt) < 0.5e18) return;
        rew.approve(address(strategyHelper), amt);
        strategyHelper.swap(address(rew), pair.token0(), haf, slippage, address(this));
        strategyHelper.swap(address(rew), pair.token1(), amt - haf, slippage, address(this));
        IERC20(pair.token0()).transfer(address(pair), IERC20(pair.token0()).balanceOf(address(this)));
        IERC20(pair.token1()).transfer(address(pair), IERC20(pair.token1()).balanceOf(address(this)));
        pair.mint(address(this));
        pair.skim(address(this));
        uint256 liq = IERC20(address(pair)).balanceOf(address(this));
        IERC20(address(pair)).approve(address(rewarder), liq);
        rewarder.deposit(poolId, liq, address(this));
        uint256 current = rate(totalShares);
        emit Earn(current, current - min(current, before));
    }

    /**
     * @notice Function to get the total managed assets
     * @return The total managed assets
     */
    function totalManagedAssets() public view returns (uint256) {
        (uint256 amt,) = rewarder.userInfo(poolId, address(this));
        return amt;
    }

    /**
     * @notice Function to exit from the strategy
     * @param strategy The address of the strategy to exit
     */
    function exit(address strategy) public auth {
        rewarder.withdraw(poolId, totalShares, strategy);
    }

    /**
     * @notice Function to move assets to a new strategy
     * @param old The address of the old strategy
     */
    function move(address old) public auth {
        require(totalShares == 0, "ts=0");
        totalShares = StrategySushiswap(old).totalShares();
        IERC20 lp = IERC20(address(pool));
        uint256 bal = lp.balanceOf(address(this));
        totalShares = bal;
        lp.approve(address(rewarder), bal);
        rewarder.deposit(poolId, bal, address(this));
    }

    /**
     * @notice Internal function to get the minimum of two values
     * @param a The first value
     * @param b The second value
     * @return The minimum value
     */
    function min(uint256 a, uint256 b) internal pure returns (uint256) {
        return a < b ? a : b;
    }

    /**
     * @notice // from OZ Math
     * @notice Internal function to calculate the square root of a value
     * @param a The value to calculate the square root for
     * @return The square root of the value
     */
    function sqrt(uint256 a) internal pure returns (uint256) {
        if (a == 0) return 0;
        uint256 result = 1 << (log2(a) >> 1);
        unchecked {
            result = (result + a / result) >> 1;
            result = (result + a / result) >> 1;
            result = (result + a / result) >> 1;
            result = (result + a / result) >> 1;
            result = (result + a / result) >> 1;
            result = (result + a / result) >> 1;
            result = (result + a / result) >> 1;
            return min(result, a / result);
        }
    }

    /**
     * @notice Internal function to calculate the log base 2 of a value
     * @param value The value to calculate the log for
     * @return The log base 2 of the value
     */
    function log2(uint256 value) internal pure returns (uint256) {
        uint256 result = 0;
        unchecked {
            if (value >> 128 > 0) {
                value >>= 128;
                result += 128;
            }
            if (value >> 64 > 0) {
                value >>= 64;
                result += 64;
            }
            if (value >> 32 > 0) {
                value >>= 32;
                result += 32;
            }
            if (value >> 16 > 0) {
                value >>= 16;
                result += 16;
            }
            if (value >> 8 > 0) {
                value >>= 8;
                result += 8;
            }
            if (value >> 4 > 0) {
                value >>= 4;
                result += 4;
            }
            if (value >> 2 > 0) {
                value >>= 2;
                result += 2;
            }
            if (value >> 1 > 0) result += 1;
        }
        return result;
    }
}

interface IERC20 {
    function symbol() external view returns (string memory);
    function decimals() external view returns (uint8);
    function balanceOf(address) external view returns (uint256);
    function approve(address, uint256) external returns (bool);
    function transfer(address, uint256) external returns (bool);
    function transferFrom(address, address, uint256) external returns (bool);
}

interface IStrategyHelper {
    function price(address) external view returns (uint256);
    function value(address, uint256) external view returns (uint256);
    function swap(address ast0, address ast1, uint256 amt, uint256 slp, address to) external returns (uint256);
}

interface IUniswapV2Pair {
    function token0() external view returns (address);
    function token1() external view returns (address);
    function totalSupply() external view returns (uint256);
    function getReserves() external view returns (uint112, uint112, uint32);
    function price0CumulativeLast() external view returns (uint256);
    function price1CumulativeLast() external view returns (uint256);
    function mint(address) external returns (uint256 liquidity);
    function burn(address) external returns (uint256 amount0, uint256 amount1);
    function swap(uint256, uint256, address, bytes calldata) external;
    function skim(address to) external;
}

interface ISushiswapMiniChefV2 {
    function SUSHI() external view returns (address);
    function lpToken(uint256) external view returns (address);
    function pendingSushi(uint256, address) external view returns (uint256);
    function userInfo(uint256, address) external view returns (uint256, int256);
    function deposit(uint256, uint256, address) external;
    function withdraw(uint256, uint256, address) external;
    function harvest(uint256, address) external;
}
