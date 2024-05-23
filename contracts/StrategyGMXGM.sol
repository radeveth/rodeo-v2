/// SPDX-License-Identifier: MIT
pragma solidity 0.8.17;

/**
 * @title StrategyGMXGM
 * @notice This contract interacts with the GMX protocol to manage investments,
 * handle deposits, withdrawals, and perform various strategy functions.
 */
contract StrategyGMXGM {
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

    /// @notice Mapping to manage execution permissions
    mapping(address => bool) public exec;

    /// @notice Mapping to manage keeper permissions
    mapping(address => bool) public keepers;

    /// @notice Exchange router contract
    IExchangeRouter public exchangeRouter;

    /// @notice Reader contract
    IReader public reader;

    /// @notice Deposit handler contract address
    address public depositHandler;

    /// @notice Withdrawal handler contract address
    address public withdrawalHandler;

    /// @notice Deposit vault contract address
    address public depositVault;

    /// @notice Withdrawal vault contract address
    address public withdrawalVault;

    /// @notice Immutable address of the data store contract
    address public immutable dataStore;

    /// @notice Immutable address of the market contract
    address public immutable market;

    /// @notice Address of the volatile token (long position)
    address public immutable tokenLong;

    /// @notice Address of the stable token (short position)
    address public immutable tokenShort;

    /// @notice Decimals for synthetic index token (optional)
    uint256 public indexTokenDecimals;

    /// @notice Amount pending for deposit
    uint256 public amountPendingDeposit;

    /// @notice Amount pending for withdrawal
    uint256 public amountPendingWithdraw;

    /// @notice Reserve ratio (in basis points)
    uint256 public reserveRatio = 1000;

    /// @notice Value for earning action (in wei)
    uint256 public earnActionValue = 0.0015675e18;

    /// @notice Gas limit for callback execution
    uint256 public callbackGasLimit = 500_000;

    /// @notice Event emitted when a configuration is changed
    event File(bytes32 indexed what, uint256 data);

    /// @notice Event emitted when a configuration address is changed
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

    /// @notice Error for invalid GMX address
    error NotGMX();

    /// @notice Error for bad token
    error BadToken();

    /// @notice Error for pending action
    error ActionPending();

    /// @notice Error for failed ETH transfer
    error ErrorSendingETH();

    /// @notice Error for wrong reserve ratio
    error WrongReserveRatio();

    /**
     * @notice Constructor to initialize the strategy contract
     * @param _asset Address of the asset token
     * @param _strategyHelper Address of the strategy helper contract
     * @param _exchangeRouter Address of the exchange router contract
     * @param _reader Address of the reader contract
     * @param _depositHandler Address of the deposit handler contract
     * @param _withdrawalHandler Address of the withdrawal handler contract
     * @param _dataStore Address of the data store contract
     * @param _market Address of the market contract
     */
    constructor(
        address _asset,
        address _strategyHelper,
        address _exchangeRouter,
        address _reader,
        address _depositHandler,
        address _withdrawalHandler,
        address _dataStore,
        address _market
    ) {
        exec[msg.sender] = true;
        asset = IERC20(_asset);
        strategyHelper = IStrategyHelper(_strategyHelper);
        exchangeRouter = IExchangeRouter(_exchangeRouter);
        reader = IReader(_reader);
        depositHandler = _depositHandler;
        withdrawalHandler = _withdrawalHandler;
        depositVault = IHandler(_depositHandler).depositVault();
        withdrawalVault = IHandler(_withdrawalHandler).withdrawalVault();
        dataStore = _dataStore;
        market = _market;

        IMarket.Props memory marketInfo = reader.getMarket(_dataStore, market);
        tokenLong = marketInfo.longToken;
        tokenShort = marketInfo.shortToken;
        name = string(
            abi.encodePacked(
                "GMX GM ", IERC20(marketInfo.longToken).symbol(), "/", IERC20(marketInfo.shortToken).symbol()
            )
        );
    }

    /**
     * @notice Fallback function to accept ETH payments
     */
    receive() external payable {}

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
        } else if (what == "reader") {
            reader = IReader(data);
        } else if (what == "exchangeRouter") {
            exchangeRouter = IExchangeRouter(data);
        } else if (what == "depositHandler") {
            depositHandler = data;
            depositVault = IHandler(data).depositVault();
        } else if (what == "withdrawalHandler") {
            withdrawalHandler = data;
            withdrawalVault = IHandler(data).withdrawalVault();
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
        } else if (what == "indexTokenDecimals") {
            indexTokenDecimals = data;
        } else if (what == "reserveRatio") {
            if (data > 10000) revert WrongReserveRatio();
            reserveRatio = data;
        } else if (what == "callbackGasLimit") {
            callbackGasLimit = data;
        } else if (what == "earnActionValue") {
            earnActionValue = data;
        } else {
            revert InvalidFile();
        }
        emit File(what, data);
    }

    /**
     * @notice Function to withdraw ETH from the contract
     */
    function withdrawEth() external auth {
        (bool success,) = address(msg.sender).call{value: address(this).balance}("");
        if (!success) revert ErrorSendingETH();
    }

    /**
     * @notice Function to withdraw airdropped tokens
     * @param token The address of the token to withdraw
     */
    function withdrawAirdrop(address token) external auth {
        if (token == address(market) || token == tokenShort || token == tokenLong) revert BadToken();
        IERC20(token).transfer(msg.sender, IERC20(token).balanceOf(address(this)));
    }

    /**
     * @notice Function to mint new shares
     * @param amount The amount of asset to mint shares for
     * @return The number of shares minted
     */
    function mint(uint256 amount) external auth loop returns (uint256) {
        uint256 slp = slippage;
        uint256 tot = totalShares;
        uint256 tma = rate(tot);

        asset.transferFrom(msg.sender, address(this), amount);
        asset.approve(address(strategyHelper), amount);
        uint256 bal = strategyHelper.swap(address(asset), tokenShort, amount, slp, address(this));
        uint256 val = strategyHelper.value(tokenShort, bal);
        uint256 shares = (tma == 0 || tot == 0) ? val : val * tot / tma;

        totalShares += shares;
        emit Mint(amount, shares);
        return shares;
    }

    /**
     * @notice Function to burn shares
     * @param shares The number of shares to burn
     * @return The amount of asset received from burning the shares
     */
    function burn(uint256 shares) external auth loop returns (uint256) {
        uint256 slp = slippage;
        uint256 val = rate(shares);
        uint256 amt = (val * (10 ** IERC20(tokenShort).decimals())) / strategyHelper.price(tokenShort);
        IERC20(tokenShort).approve(address(strategyHelper), amt);
        uint256 bal = strategyHelper.swap(tokenShort, address(asset), amt, slp, msg.sender);

        totalShares -= shares;
        emit Burn(bal, shares);
        return bal;
    }

    /**
     * @notice Function to kill shares
     * @param shares The number of shares to kill
     * @param to The address to send the assets to
     * @return The data from the kill action
     */
    function kill(uint256 shares, address to) external auth loop returns (bytes memory) {
        uint256 value = rate(shares);
        uint256 amount = (value * (10 ** IERC20(tokenShort).decimals())) / strategyHelper.price(tokenShort);
        IERC20(tokenShort).transfer(to, amount);

        totalShares -= shares;
        emit Kill(amount, shares);

        address[] memory assets = new address[](1);
        assets[0] = tokenShort;
        return abi.encode(bytes32("gmxgm"), assets);
    }

    /**
     * @notice Function to perform earning actions
     */
    function earn() external payable loop {
        if (!keepers[msg.sender]) revert NotKeeper();
        uint256 before = rate(totalShares);

        if (amountPendingDeposit != 0 || amountPendingWithdraw != 0) {
            revert ActionPending();
        }

        uint256 slp = slippage;
        uint256 bal = IERC20(tokenLong).balanceOf(address(this));
        if (bal > 0) {
            IERC20(tokenLong).approve(address(strategyHelper), bal);
            strategyHelper.swap(tokenLong, tokenShort, bal, slp, address(this));
        }

        bal = IERC20(tokenShort).balanceOf(address(this));
        uint256 have = strategyHelper.value(tokenShort, bal);
        uint256 need = rate(totalShares) * reserveRatio / 10000;

        if (have > need) {
            uint256 amt = (have - need) * bal / have;
            uint256 haf = amt / 2;
            IERC20(tokenShort).approve(address(strategyHelper), haf);
            uint256 out = strategyHelper.swap(tokenShort, tokenLong, haf, slp, address(this));
            uint256 minOut = (have - need) * 1e18 / marketTokenPrice(true);
            amountPendingDeposit = minOut;
            minOut = minOut * (10000 - slp) / 10000;

            IExchangeRouter.CreateDepositParams memory params = IExchangeRouter.CreateDepositParams({
                receiver: address(this),
                callbackContract: address(this),
                uiFeeReceiver: address(0),
                market: market,
                initialLongToken: tokenLong,
                initialShortToken: tokenShort,
                longTokenSwapPath: new address[](0),
                shortTokenSwapPath: new address[](0),
                minMarketTokens: minOut,
                shouldUnwrapNativeToken: false,
                executionFee: earnActionValue,
                callbackGasLimit: callbackGasLimit
            });

            IMarket.Props memory marketInfo = reader.getMarket(dataStore, market);
            bytes[] memory data = new bytes[](4);
            address router = exchangeRouter.router();
            address vault = depositVault;

            IERC20(marketInfo.longToken).approve(router, out);
            IERC20(marketInfo.shortToken).approve(router, amt - haf);

            data[0] = abi.encodeWithSelector(IExchangeRouter.sendWnt.selector, vault, params.executionFee);
            data[1] = abi.encodeWithSelector(IExchangeRouter.sendTokens.selector, marketInfo.longToken, vault, out);
            data[2] =
                abi.encodeWithSelector(IExchangeRouter.sendTokens.selector, marketInfo.shortToken, vault, amt - haf);
            data[3] = abi.encodeWithSelector(IExchangeRouter.createDeposit.selector, params);
            exchangeRouter.multicall{value: params.executionFee}(data);
        } else if (have < need) {
            uint256 amt = (need - have) * 1e18 / marketTokenPrice(true);

            IMarket.Props memory marketInfo = reader.getMarket(dataStore, market);
            (uint256 longOut, uint256 shortOut) = reader.getWithdrawalAmountOut(
                dataStore,
                marketInfo,
                IMarket.Prices({
                    indexTokenPrice: gmxPrice(marketInfo.indexToken, true),
                    longTokenPrice: gmxPrice(marketInfo.longToken, false),
                    shortTokenPrice: gmxPrice(marketInfo.shortToken, false)
                }),
                amt,
                address(0)
            );
            IExchangeRouter.CreateWithdrawalParams memory params = IExchangeRouter.CreateWithdrawalParams({
                receiver: address(this),
                callbackContract: address(this),
                uiFeeReceiver: address(0),
                market: market,
                longTokenSwapPath: new address[](0),
                shortTokenSwapPath: new address[](0),
                minLongTokenAmount: longOut * (10000 - slp) / 10000,
                minShortTokenAmount: shortOut * (10000 - slp) / 10000,
                shouldUnwrapNativeToken: false,
                executionFee: earnActionValue,
                callbackGasLimit: callbackGasLimit
            });

            amountPendingWithdraw = amt;
            bytes[] memory data = new bytes[](3);
            IERC20(market).approve(exchangeRouter.router(), amt);
            data[0] = abi.encodeWithSelector(IExchangeRouter.sendWnt.selector, withdrawalVault, params.executionFee);
            data[1] = abi.encodeWithSelector(IExchangeRouter.sendTokens.selector, market, withdrawalVault, amt);
            data[2] = abi.encodeWithSelector(IExchangeRouter.createWithdrawal.selector, params);
            exchangeRouter.multicall{value: params.executionFee}(data);
        }

        uint256 current = rate(totalShares);
        emit Earn(current, current - min(current, before));
    }

    /**
     * @notice Function to exit from the strategy
     * @param strategy The address of the strategy to exit
     */
    function exit(address strategy) external auth {
        if (amountPendingDeposit != 0 || amountPendingWithdraw != 0) {
            revert ActionPending();
        }
        IERC20(market).transfer(strategy, IERC20(market).balanceOf(address(this)));
        IERC20(tokenLong).transfer(strategy, IERC20(tokenLong).balanceOf(address(this)));
        IERC20(tokenShort).transfer(strategy, IERC20(tokenShort).balanceOf(address(this)));
    }

    /**
     * @notice Function to move assets to a new strategy
     * @param old The address of the old strategy
     */
    function move(address old) external auth {}

    /**
     * @notice Function to calculate the rate of shares
     * @param shares The number of shares
     * @return The rate of the shares
     */
    function rate(uint256 shares) public view returns (uint256) {
        uint256 val = strategyHelper.value(tokenLong, IERC20(tokenLong).balanceOf(address(this)));
        val += strategyHelper.value(tokenShort, IERC20(tokenShort).balanceOf(address(this)));

        uint256 bal = IERC20(market).balanceOf(address(this)) + amountPendingDeposit + amountPendingWithdraw;
        val += bal * marketTokenPrice(true) / 1e18;

        return shares * val / totalShares;
    }

    /**
     * @notice Function to get the market token price
     * @param isDeposit Whether the price is for a deposit
     * @return The market token price
     */
    function marketTokenPrice(bool isDeposit) public view returns (uint256) {
        IReader r = reader;
        address store = dataStore;
        IMarket.Props memory marketInfo = r.getMarket(store, market);
        (int256 price,) = r.getMarketTokenPrice(
            store,
            marketInfo,
            gmxPrice(marketInfo.indexToken, true),
            gmxPrice(marketInfo.longToken, false),
            gmxPrice(marketInfo.shortToken, false),
            isDeposit ? MAX_PNL_FACTOR_FOR_DEPOSITS : MAX_PNL_FACTOR_FOR_WITHDRAWALS,
            false // maximize
        );

        // returned as 1e30, lets downscale to 1e18 used in here
        return price < 0 ? 0 : uint256(price) / 1e12;
    }

    /// @notice Constant for max PnL factor for deposits
    bytes32 constant MAX_PNL_FACTOR_FOR_DEPOSITS = keccak256(abi.encode("MAX_PNL_FACTOR_FOR_DEPOSITS"));

    /// @notice Constant for max PnL factor for withdrawals
    bytes32 constant MAX_PNL_FACTOR_FOR_WITHDRAWALS = keccak256(abi.encode("MAX_PNL_FACTOR_FOR_WITHDRAWALS"));

    /**
     * @notice Internal function to get the GMX price
     * @param token The address of the token
     * @param isIndex Whether the token is an index token
     * @return The price properties of the token
     */
    function gmxPrice(address token, bool isIndex) internal view returns (IPrice.Props memory) {
        uint256 decimals = isIndex ? indexTokenDecimals : 0;
        if (decimals == 0) {
            decimals = IERC20(token).decimals();
        }
        uint256 price = strategyHelper.price(token);
        price = price * (10 ** (30 - decimals)) / 1e18;
        return IPrice.Props({min: price, max: price});
    }

    /**
     * @notice Function to handle deposit execution
     * @param props The properties of the deposit
     * @param eventLog The event log data
     */
    function afterDepositExecution(bytes32, IDeposit.Props memory props, IEventUtils.EventLogData memory eventLog)
        external
    {
        if (msg.sender != depositHandler) revert NotGMX();
        amountPendingDeposit = 0;
    }

    /**
     * @notice Function to handle deposit cancellation
     * @param props The properties of the deposit
     * @param eventLog The event log data
     */
    function afterDepositCancellation(bytes32, IDeposit.Props memory props, IEventUtils.EventLogData memory eventLog)
        external
    {
        if (msg.sender != depositHandler) revert NotGMX();
        amountPendingDeposit = 0;
    }

    /**
     * @notice Function to handle withdrawal execution
     */
    function afterWithdrawalExecution(
        bytes32,
        IWithdrawal.Props memory, /*props */
        IEventUtils.EventLogData memory /* eventLog */
    ) external {
        if (msg.sender != withdrawalHandler) revert NotGMX();
        amountPendingWithdraw = 0;
        uint256 bal = IERC20(tokenLong).balanceOf(address(this));
        IERC20(tokenLong).approve(address(strategyHelper), bal);
        try strategyHelper.swap(tokenLong, tokenShort, bal, slippage, address(this)) {} catch {}
    }

    /**
     * @notice Function to handle withdrawal cancellation
     */
    function afterWithdrawalCancellation(
        bytes32,
        IWithdrawal.Props memory, /* props */
        IEventUtils.EventLogData memory /* eventLog */
    ) external {
        if (msg.sender != withdrawalHandler) revert NotGMX();
        amountPendingWithdraw = 0;
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
}

interface IERC20 {
    function symbol() external view returns (string memory);
    function decimals() external view returns (uint8);
    function balanceOf(address) external view returns (uint256);
    function approve(address, uint256) external;
    function transfer(address, uint256) external;
    function transferFrom(address, address, uint256) external;
}

interface IStrategyHelper {
    function price(address) external view returns (uint256);
    function value(address, uint256) external view returns (uint256);
    function swap(address ast0, address ast1, uint256 amt, uint256 slp, address to) external returns (uint256);
}

interface IHandler {
    function depositVault() external view returns (address);
    function withdrawalVault() external view returns (address);
}

interface IPriceFeed {
    function latestRoundData()
        external
        view
        returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound);
}

interface IDataStore {
    function getUint(bytes32 key) external view returns (uint256);
    function getAddress(bytes32 key) external view returns (address);
    function getBytes32(bytes32 key) external view returns (bytes32);
}

interface IExchangeRouter {
    struct CreateDepositParams {
        address receiver;
        address callbackContract;
        address uiFeeReceiver;
        address market;
        address initialLongToken;
        address initialShortToken;
        address[] longTokenSwapPath;
        address[] shortTokenSwapPath;
        uint256 minMarketTokens;
        bool shouldUnwrapNativeToken;
        uint256 executionFee;
        uint256 callbackGasLimit;
    }

    struct CreateWithdrawalParams {
        address receiver;
        address callbackContract;
        address uiFeeReceiver;
        address market;
        address[] longTokenSwapPath;
        address[] shortTokenSwapPath;
        uint256 minLongTokenAmount;
        uint256 minShortTokenAmount;
        bool shouldUnwrapNativeToken;
        uint256 executionFee;
        uint256 callbackGasLimit;
    }

    function sendWnt(address receiver, uint256 amount) external payable;
    function sendTokens(address token, address receiver, uint256 amount) external payable;
    function createDeposit(CreateDepositParams calldata params) external payable returns (bytes32);
    function createWithdrawal(CreateWithdrawalParams calldata params) external payable returns (bytes32);
    function multicall(bytes[] calldata data) external payable returns (bytes[] memory results);
    function router() external returns (address);
}

interface IMarket {
    struct Prices {
        IPrice.Props indexTokenPrice;
        IPrice.Props longTokenPrice;
        IPrice.Props shortTokenPrice;
    }

    struct Props {
        address marketToken;
        address indexToken;
        address longToken;
        address shortToken;
    }

    function mint(address account, uint256 amount) external;
}

interface IPrice {
    struct Props {
        uint256 min;
        uint256 max;
    }
}

interface IMarketPoolValueInfo {
    struct Props {
        int256 poolValue;
        int256 longPnl;
        int256 shortPnl;
        int256 netPnl;
        uint256 longTokenAmount;
        uint256 shortTokenAmount;
        uint256 longTokenUsd;
        uint256 shortTokenUsd;
        uint256 totalBorrowingFees;
        uint256 borrowingFeePoolFactor;
        uint256 impactPoolAmount;
    }
}

interface IReader {
    function getMarket(address dataStore, address key) external view returns (IMarket.Props memory);
    function getMarketTokenPrice(
        address dataStore,
        IMarket.Props memory market,
        IPrice.Props memory indexTokenPrice,
        IPrice.Props memory longTokenPrice,
        IPrice.Props memory shortTokenPrice,
        bytes32 pnlFactorType,
        bool maximize
    ) external view returns (int256, IMarketPoolValueInfo.Props memory);
    function getWithdrawalAmountOut(
        address dataStore,
        IMarket.Props memory market,
        IMarket.Prices memory prices,
        uint256 marketTokenAmount,
        address uiFeeReceiver
    ) external view returns (uint256, uint256);
}

interface IDeposit {
    struct Props {
        Addresses addresses;
        Numbers numbers;
        Flags flags;
    }

    struct Addresses {
        address account;
        address receiver;
        address callbackContract;
        address uiFeeReceiver;
        address market;
        address initialLongToken;
        address initialShortToken;
        address[] longTokenSwapPath;
        address[] shortTokenSwapPath;
    }

    struct Numbers {
        uint256 initialLongTokenAmount;
        uint256 initialShortTokenAmount;
        uint256 minMarketTokens;
        uint256 updatedAtBlock;
        uint256 executionFee;
        uint256 callbackGasLimit;
    }

    struct Flags {
        bool shouldUnwrapNativeToken;
    }
}

interface IWithdrawal {
    struct Props {
        Addresses addresses;
        Numbers numbers;
        Flags flags;
    }

    struct Addresses {
        address account;
        address receiver;
        address callbackContract;
        address uiFeeReceiver;
        address market;
        address[] longTokenSwapPath;
        address[] shortTokenSwapPath;
    }

    struct Numbers {
        uint256 marketTokenAmount;
        uint256 minLongTokenAmount;
        uint256 minShortTokenAmount;
        uint256 updatedAtBlock;
        uint256 executionFee;
        uint256 callbackGasLimit;
    }

    struct Flags {
        bool shouldUnwrapNativeToken;
    }
}

interface IEventUtils {
    struct EventLogData {
        AddressItems addressItems;
        UintItems uintItems;
        IntItems intItems;
        BoolItems boolItems;
        Bytes32Items bytes32Items;
        BytesItems bytesItems;
        StringItems stringItems;
    }

    struct AddressItems {
        AddressKeyValue[] items;
        AddressArrayKeyValue[] arrayItems;
    }

    struct UintItems {
        UintKeyValue[] items;
        UintArrayKeyValue[] arrayItems;
    }

    struct IntItems {
        IntKeyValue[] items;
        IntArrayKeyValue[] arrayItems;
    }

    struct BoolItems {
        BoolKeyValue[] items;
        BoolArrayKeyValue[] arrayItems;
    }

    struct Bytes32Items {
        Bytes32KeyValue[] items;
        Bytes32ArrayKeyValue[] arrayItems;
    }

    struct BytesItems {
        BytesKeyValue[] items;
        BytesArrayKeyValue[] arrayItems;
    }

    struct StringItems {
        StringKeyValue[] items;
        StringArrayKeyValue[] arrayItems;
    }

    struct AddressKeyValue {
        string key;
        address value;
    }

    struct AddressArrayKeyValue {
        string key;
        address[] value;
    }

    struct UintKeyValue {
        string key;
        uint256 value;
    }

    struct UintArrayKeyValue {
        string key;
        uint256[] value;
    }

    struct IntKeyValue {
        string key;
        int256 value;
    }

    struct IntArrayKeyValue {
        string key;
        int256[] value;
    }

    struct BoolKeyValue {
        string key;
        bool value;
    }

    struct BoolArrayKeyValue {
        string key;
        bool[] value;
    }

    struct Bytes32KeyValue {
        string key;
        bytes32 value;
    }

    struct Bytes32ArrayKeyValue {
        string key;
        bytes32[] value;
    }

    struct BytesKeyValue {
        string key;
        bytes value;
    }

    struct BytesArrayKeyValue {
        string key;
        bytes[] value;
    }

    struct StringKeyValue {
        string key;
        string value;
    }

    struct StringArrayKeyValue {
        string key;
        string[] value;
    }
}
