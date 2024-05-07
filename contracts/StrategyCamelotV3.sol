// SPDX-License-Identifier: MIT
pragma solidity 0.8.17;

import {TickMath} from "./vendor/TickMath.sol";
import {LiquidityAmounts} from "./vendor/LiquidityAmounts.sol";

contract StrategyCamelotV3 {
    string public name;
    uint256 public totalShares = 1_000_000;
    uint256 public slippage = 500;
    bool internal entered;
    IERC20 public asset;
    IStrategyHelper public strategyHelper;
    mapping(address => bool) public exec;
    mapping(address => bool) public keepers;

    IXGrail public immutable xgrail;
    IStrategyHelperUniswapV3 public immutable strategyHelperUniswapV3;
    IUniProxy public immutable uniProxy;
    IQuoter public immutable quoter;
    IHypervisor public immutable hypervisor;
    bytes public pathToLp; // UniV3 path from targetAsset to other asset
    address public targetAsset;
    INFTPool public nftPool;
    INitroPool public nitroPool;
    uint256 public tokenId;
    uint32 public twapPeriod = 43200;
    address public rewardToken1;
    address public rewardToken2;
    address public rewardToken3;

    event File(bytes32 indexed what, uint256 data);
    event File(bytes32 indexed what, address data);
    event Mint(uint256 amount, uint256 shares);
    event Burn(uint256 amount, uint256 shares);
    event Kill(uint256 amount, uint256 shares);
    event Earn(uint256 tvl, uint256 profit);

    error NotKeeper();
    error InvalidFile();
    error NoReentering();
    error Unauthorized();
    error PriceSlipped();
    error WrongTargetAsset();
    error TwapPeriodTooLong();
    error TokenIdNeededFirst();

    constructor(
        address _asset,
        address _strategyHelper,
        address _xgrail,
        address _strategyHelperUniswapV3,
        address _uniProxy,
        address _quoter,
        address _hypervisor,
        address _nftPool,
        address _targetAsset,
        bytes memory _pathToLp
    ) {
        exec[msg.sender] = true;
        asset = IERC20(_asset);
        strategyHelper = IStrategyHelper(_strategyHelper);
        xgrail = IXGrail(_xgrail);
        strategyHelperUniswapV3 = IStrategyHelperUniswapV3(_strategyHelperUniswapV3);
        uniProxy = IUniProxy(_uniProxy);
        quoter = IQuoter(_quoter);
        hypervisor = IHypervisor(_hypervisor);
        nftPool = INFTPool(_nftPool);
        targetAsset = _targetAsset;
        pathToLp = _pathToLp;
        name = string(abi.encodePacked("Camelot V3 ", hypervisor.token0().symbol(), "/", hypervisor.token1().symbol()));

        if (_targetAsset != address(hypervisor.token0()) && _targetAsset != address(hypervisor.token1())) {
            revert WrongTargetAsset();
        }
    }

    modifier loop() {
        if (entered) revert NoReentering();
        entered = true;
        _;
        entered = false;
    }

    modifier auth() {
        if (!exec[msg.sender]) revert Unauthorized();
        _;
    }

    function file(bytes32 what, address data) external auth {
        if (what == "exec") {
            exec[data] = !exec[data];
        } else if (what == "keeper") {
            keepers[data] = !keepers[data];
        } else if (what == "rewardToken1") {
            rewardToken1 = data;
        } else if (what == "rewardToken2") {
            rewardToken2 = data;
        } else if (what == "rewardToken3") {
            rewardToken3 = data;
        } else {
            revert InvalidFile();
        }
        emit File(what, data);
    }

    function file(bytes32 what, uint256 data) external auth {
        if (what == "slippage") {
            slippage = data;
        } else if (what == "twapPeriod") {
            if (data > uint256(int256(type(int32).max))) revert TwapPeriodTooLong();
            twapPeriod = uint32(twapPeriod);
        } else {
            revert InvalidFile();
        }
        emit File(what, data);
    }

    function setPathToLp(bytes calldata newPathToLp) external auth {
        pathToLp = newPathToLp;
    }

    function setNitroPool(address _nitroPool) external auth {
        if (tokenId == 0) revert TokenIdNeededFirst();
        if (_nitroPool == address(0)) {
            nitroPool.withdraw(tokenId);
        } else {
            nftPool.safeTransferFrom(address(this), _nitroPool, tokenId, "");
        }
        nitroPool = INitroPool(_nitroPool);
    }

    function xgrailRedeem(uint256 amount, uint256 duration) external auth {
        xgrail.redeem(amount, duration);
    }

    function xgrailFinalizeRedeem(uint256 index) external auth {
        xgrail.finalizeRedeem(index);
    }

    function mint(uint256 amount) external auth loop returns (uint256) {
        asset.transferFrom(msg.sender, address(this), amount);
        address tgtAst = targetAsset;
        uint256 slp = slippage;
        uint256 tma = totalManagedAssets();

        asset.approve(address(strategyHelper), amount);
        strategyHelper.swap(address(asset), tgtAst, amount, slp, address(this));

        uint256 liq;
        {
            uint256 tgtAmt = IERC20(tgtAst).balanceOf(address(this));
            (uint256 amt0, uint256 amt1) = quoteAndSwap(tgtAst, tgtAmt, slp);
            address hyp = address(hypervisor);
            hypervisor.token0().approve(hyp, amt0);
            hypervisor.token1().approve(hyp, amt1);
            liq = uniProxy.deposit(amt0, amt1, address(this), hyp, [uint256(0), 0, 0, 0]);
            stake(liq);
        }

        uint256 val = valueLiquidity() * liq / totalManagedAssets();
        if (val < strategyHelper.value(address(asset), amount) * (10000 - slp) / 10000) revert PriceSlipped();
        uint256 shares = tma == 0 ? liq : liq * totalShares / tma;

        totalShares += shares;
        emit Mint(amount, shares);
        return shares;
    }

    function burn(uint256 shares) external auth loop returns (uint256) {
        uint256 tma = totalManagedAssets();
        uint256 amt = (shares * tma) / totalShares;
        uint256 val = valueLiquidity() * amt / tma;
        unstake(amt);
        (uint256 amt0, uint256 amt1) = hypervisor.withdraw(amt, address(this), address(this), [uint256(0), 0, 0, 0]);

        address strategyHelperAddress = address(strategyHelper);
        hypervisor.token0().approve(strategyHelperAddress, amt0);
        hypervisor.token1().approve(strategyHelperAddress, amt1);

        uint256 bal;
        uint256 slp = slippage;
        bal += strategyHelper.swap(address(hypervisor.token0()), address(asset), amt0, slp, msg.sender);
        bal += strategyHelper.swap(address(hypervisor.token1()), address(asset), amt1, slp, msg.sender);

        if (strategyHelper.value(address(asset), bal) < val * (10000 - slp) / 10000) revert PriceSlipped();

        totalShares -= shares;
        emit Burn(bal, shares);
        return bal;
    }

    function kill(uint256 shares, address to) external auth loop returns (bytes memory) {
        uint256 amount = shares * totalManagedAssets() / totalShares;
        unstake(amount);
        hypervisor.transfer(to, amount);

        totalShares -= shares;
        emit Kill(amount, shares);

        address[] memory assets = new address[](1);
        assets[0] = address(hypervisor);
        return abi.encode(bytes32("camelotv3"), assets);
    }

    function stake(uint256 amount) internal {
        IERC20(address(hypervisor)).approve(address(nftPool), amount);
        if (tokenId != 0 && totalManagedAssets() > 0) {
            nftPool.addToPosition(tokenId, amount);
        } else {
            // Clear the token if it's already set from a position that
            // went to 0 and got burned
            if (tokenId != 0) tokenId = 0;
            nftPool.createPosition(amount, 0);
        }
    }

    function unstake(uint256 amount) internal {
        if (address(nitroPool) != address(0)) {
            nitroPool.withdraw(tokenId);
        }
        nftPool.withdrawFromPosition(tokenId, amount);
        if (address(nitroPool) != address(0)) {
            nftPool.safeTransferFrom(address(this), address(nitroPool), tokenId, "");
        }
    }

    function earn() external payable loop {
        if (!keepers[msg.sender]) revert NotKeeper();
        uint256 before = rate(totalShares);

        if (tokenId == 0) return;
        uint256 slp = slippage;
        address tgtAsset = targetAsset;
        nftPool.harvestPosition(tokenId);
        if (address(nitroPool) != address(0)) {
            nitroPool.harvest();
        }

        if (rewardToken1 != address(0)) {
            uint256 balance = IERC20(rewardToken1).balanceOf(address(this));
            if (strategyHelper.value(rewardToken1, balance) > 1e18) {
                IERC20(rewardToken1).approve(address(strategyHelper), balance);
                strategyHelper.swap(rewardToken1, tgtAsset, balance, slp, address(this));
            }
        }

        if (rewardToken2 != address(0)) {
            uint256 balance = IERC20(rewardToken2).balanceOf(address(this));
            if (strategyHelper.value(rewardToken2, balance) > 1e18) {
                IERC20(rewardToken2).approve(address(strategyHelper), balance);
                strategyHelper.swap(rewardToken2, tgtAsset, balance, slp, address(this));
            }
        }

        if (rewardToken3 != address(0)) {
            uint256 balance = IERC20(rewardToken3).balanceOf(address(this));
            if (strategyHelper.value(rewardToken3, balance) > 1e18) {
                IERC20(rewardToken3).approve(address(strategyHelper), balance);
                strategyHelper.swap(rewardToken3, tgtAsset, balance, slp, address(this));
            }
        }

        uint256 amt = IERC20(tgtAsset).balanceOf(address(this));
        if (strategyHelper.value(tgtAsset, amt) < 1e18) return;
        (uint256 amt0, uint256 amt1) = quoteAndSwap(tgtAsset, amt, slp);
        address h = address(hypervisor);
        IHypervisor(h).token0().approve(h, amt0);
        IHypervisor(h).token1().approve(h, amt1);
        stake(uniProxy.deposit(amt0, amt1, address(this), h, [uint256(0), 0, 0, 0]));

        uint256 current = rate(totalShares);
        emit Earn(current, current - min(current, before));
    }

    function exit(address strategy) external auth {
        if (tokenId == 0) return;
        if (address(nitroPool) != address(0)) {
            nitroPool.withdraw(tokenId);
        }
        nftPool.safeTransferFrom(address(this), strategy, tokenId, "");
    }

    function move(address old) external auth {
        nftPool = StrategyCamelotV3(old).nftPool();
        nitroPool = StrategyCamelotV3(old).nitroPool();
        tokenId = StrategyCamelotV3(old).tokenId();
        if (address(nitroPool) != address(0)) {
            nftPool.safeTransferFrom(address(this), address(nitroPool), tokenId, "");
        }
    }

    function rate(uint256 shares) public view returns (uint256) {
        return shares * valueLiquidity() / totalShares;
    }

    function quoteAddLiquidity(uint256 amt, address trgtAst, bytes memory path) private returns (uint256, uint256) {
        uint256 lp0Amt = amt / 2;
        uint256 lp1Amt = amt - lp0Amt;
        uint256 out0 = lp0Amt;
        uint256 out1 = lp1Amt;
        bytes memory path0 = trgtAst != address(hypervisor.token0()) ? path : bytes("");
        bytes memory path1 = trgtAst != address(hypervisor.token1()) ? path : bytes("");

        if (path0.length > 0) {
            out0 = quoter.quoteExactInput(path0, lp0Amt);
        }
        if (path1.length > 0) {
            out1 = quoter.quoteExactInput(path1, lp1Amt);
        }

        (uint256 start, uint256 end) =
            uniProxy.getDepositAmount(address(hypervisor), address(hypervisor.token0()), out0);
        uint256 toLp0 = amt * 1e18 / ((((start + end) / 2) * 1e18 / out1) + 1e18);
        uint256 toLp1 = amt - toLp0;

        return (toLp0, toLp1);
    }

    function quoteAndSwap(address trgtAst, uint256 amt, uint256 slp) private returns (uint256 amt0, uint256 amt1) {
        bytes memory path = pathToLp;
        (uint256 toLp0, uint256 toLp1) = quoteAddLiquidity(amt, trgtAst, path);
        address token0 = address(hypervisor.token0());

        if (trgtAst == token0) {
            uint256 before = hypervisor.token1().balanceOf(address(this));
            swap(trgtAst, address(hypervisor.token1()), path, toLp1, slp);
            amt0 = toLp0;
            amt1 = hypervisor.token1().balanceOf(address(this)) - before;
        } else {
            uint256 before = hypervisor.token0().balanceOf(address(this));
            swap(trgtAst, token0, path, toLp0, slp);
            amt0 = hypervisor.token0().balanceOf(address(this)) - before;
            amt1 = toLp1;
        }
    }

    function swap(address trgtAst, address ast, bytes memory path, uint256 toLp, uint256 slp) private {
        uint256 minOut = strategyHelper.convert(trgtAst, ast, toLp) * (10000 - slp) / 10000;
        IERC20(trgtAst).transfer(address(strategyHelperUniswapV3), toLp);
        strategyHelperUniswapV3.swap(trgtAst, path, toLp, minOut, address(this));
    }

    function valueLiquidity() private view returns (uint256) {
        uint32 period = twapPeriod;
        uint32[] memory secondsAgos = new uint32[](2);

        secondsAgos[0] = period;
        secondsAgos[1] = 0;

        (int56[] memory tickCumulatives,,,) = hypervisor.pool().getTimepoints(secondsAgos);
        uint160 midX96 = TickMath.getSqrtRatioAtTick(int24((tickCumulatives[1] - tickCumulatives[0]) / int32(period)));
        (uint256 bas0, uint256 bas1) = getPosition(midX96, hypervisor.baseLower(), hypervisor.baseUpper());
        (uint256 lim0, uint256 lim1) = getPosition(midX96, hypervisor.limitLower(), hypervisor.limitUpper());
        uint256 val0 = strategyHelper.value(
            address(hypervisor.token0()), bas0 + lim0 + hypervisor.token0().balanceOf(address(hypervisor))
        );
        uint256 val1 = strategyHelper.value(
            address(hypervisor.token1()), bas1 + lim1 + hypervisor.token1().balanceOf(address(hypervisor))
        );
        uint256 bal = totalManagedAssets();
        uint256 spl = hypervisor.totalSupply();

        val0 = val0 * bal / spl;
        val1 = val1 * bal / spl;

        return val0 + val1;
    }

    function totalManagedAssets() private view returns (uint256) {
        if (tokenId == 0) return 0;
        (uint256 amount,,,,,,,) = nftPool.getStakingPosition(tokenId);
        return amount;
    }

    function getPosition(uint160 midX96, int24 minTick, int24 maxTick) private view returns (uint256, uint256) {
        bytes32 key;
        address owner = address(hypervisor);
        assembly {
            key := or(shl(24, or(shl(24, owner), and(minTick, 0xFFFFFF))), and(maxTick, 0xFFFFFF))
        }
        (uint128 liq,,,, uint128 owed0, uint128 owed1) = IAlgebraPool(address(hypervisor.pool())).positions(key);
        (uint256 amt0, uint256 amt1) = LiquidityAmounts.getAmountsForLiquidity(
            midX96, TickMath.getSqrtRatioAtTick(minTick), TickMath.getSqrtRatioAtTick(maxTick), liq
        );

        return (amt0 + uint256(owed0), amt1 + uint256(owed1));
    }

    function onERC721Received(address, address, uint256 _tokenId, bytes calldata) external returns (bytes4) {
        if (msg.sender == address(nftPool) && tokenId == 0) {
            tokenId = _tokenId;
        }
        return StrategyCamelotV3.onERC721Received.selector;
    }

    function onNFTHarvest(address, address, uint256, uint256, uint256) public pure returns (bool) {
        return true;
    }

    function onNFTAddToPosition(address, uint256, uint256) public pure returns (bool) {
        return true;
    }

    function onNFTWithdraw(address, uint256, uint256) public pure returns (bool) {
        return true;
    }

    function min(uint256 a, uint256 b) internal pure returns (uint256) {
        return a < b ? a : b;
    }
}

interface IERC20 {
    function symbol() external view returns (string memory);
    function balanceOf(address) external view returns (uint256);
    function approve(address, uint256) external;
    function transfer(address, uint256) external;
    function transferFrom(address, address, uint256) external;
}

interface IStrategyHelper {
    function value(address, uint256) external view returns (uint256);
    function convert(address, address, uint256) external view returns (uint256);
    function swap(address ast0, address ast1, uint256 amt, uint256 slp, address to) external returns (uint256);
}

interface IStrategyHelperUniswapV3 {
    function swap(address ast, bytes calldata path, uint256 amt, uint256 min, address to) external;
}

interface IUniProxy {
    function deposit(uint256 deposit0, uint256 deposit1, address to, address pos, uint256[4] memory minIn) external returns (uint256 shares);
    function getDepositAmount(address pos, address token, uint256 deposit) external view returns (uint256 amountStart, uint256 amountEnd);
}

interface IQuoter {
    function quoteExactInput(bytes memory path, uint256 amountIn) external returns (uint256 amountOut);
}

interface IHypervisor is IERC20 {
    function withdraw(uint256 shares, address to, address from, uint256[4] memory minAmounts) external returns (uint256 amount0, uint256 amount1);
    function pool() external view returns (IAlgebraPool);
    function token0() external view returns (IERC20);
    function token1() external view returns (IERC20);
    function totalSupply() external view returns (uint256);
    function baseLower() external view returns (int24);
    function baseUpper() external view returns (int24);
    function limitLower() external view returns (int24);
    function limitUpper() external view returns (int24);
}

interface INFTPool {
    function safeTransferFrom(address from, address to, uint256 tokenId, bytes memory _data) external;
    function createPosition(uint256 amount, uint256 lockDuration) external;
    function addToPosition(uint256 tokenId, uint256 amountToAdd) external;
    function withdrawFromPosition(uint256 tokenId, uint256 amountToWithdraw) external;
    function harvestPosition(uint256 tokenId) external;
    function getStakingPosition(uint256 tokenId) external view returns (
        uint256 amount, uint256 amountWithMultiplier, uint256 startLockTime,
        uint256 lockDuration, uint256 lockMultiplier, uint256 rewardDebt,
        uint256 boostPoints, uint256 totalMultiplier
    );
}

interface INitroPool {
    function harvest() external;
    function withdraw(uint256 tokenId) external;
}

interface IAlgebraPool {
    function token0() external view returns (address);
    function token1() external view returns (address);
    function getTimepoints(uint32[] calldata secondsAgos) external view returns (
        int56[] memory tickCumulatives,
        uint160[] memory secondsPerLiquidityCumulatives,
        uint112[] memory volatilityCumulatives,
        uint256[] memory volumePerAvgLiquiditys
    );
    function positions(bytes32 key) external view returns (
        uint128 liquidityAmount,
        uint32 lastLiquidityAddTimestamp,
        uint256 innerFeeGrowth0Token,
        uint256 innerFeeGrowth1Token,
        uint128 fees0,
        uint128 fees1
    );
}

interface IXGrail {
  function redeem(uint256 xGrailAmount, uint256 duration) external;
  function finalizeRedeem(uint256 redeemIndex) external;
}
