// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IERC20 } from "./interfaces/IERC20.sol";
import { IInvestor } from "./interfaces/IInvestor.sol";
import { IAavePool } from "./interfaces/IAavePool.sol";
import { IStrategyHelper } from "./interfaces/IStrategyHelper.sol";
import { ISushi } from "./interfaces/ISushi.sol";

/// @title Helper Contract for Rodeo V2 Finance
/// @notice This contract provides various helper functions for interacting with investment strategies and handling liquidations.
/// @dev The contract uses flash loans from Aave and integrates with SushiSwap for asset swaps.
contract Helper {
    IInvestor public investor;
    IERC20 public asset;
    IAavePool public lender;
    IStrategyHelper public sh;
    uint256 public slippage = 2500;

    /// @notice Constructor to initialize the Helper contract
    /// @param _investor Address of the Investor contract
    /// @param _asset Address of the ERC20 asset used for transactions
    /// @param _lender Address of the Aave lending pool
    /// @param _sh Address of the Strategy Helper contract
    constructor(address _investor, address _asset, address _lender, address _sh) {
        investor = IInvestor(_investor);
        asset = IERC20(_asset);
        lender = IAavePool(_lender);
        sh = IStrategyHelper(_sh);
    }

    function setInvestor(address _investor) external {
        investor = IInvestor(_investor);
    }

    /// @notice Kills an undercollateralized position using a flash loan
    /// @param id The ID of the position to be killed
    function kill(uint256 id) external {
        uint256 repay = investor.killRepayment(id);
        lender.flashLoanSimple(address(this), address(asset), repay, abi.encodePacked(id), 0);
    }

    function partialKill(uint256 id, uint256 amount) external {
        uint256 repay = investor.partialKillRepayment(amount);
        lender.flashLoanSimple(address(this), address(asset), repay, abi.encodePacked(id), 0);
    }

    /// @notice Executes the flash loan operation
    /// @param amount The amount of the flash loan
    /// @param premium The premium fee for the flash loan
    /// @param initiator The address initiating the flash loan
    /// @param params Additional parameters passed to the flash loan
    /// @return A boolean indicating the success of the operation
    function executeOperation(address, uint256 amount, uint256 premium, address initiator, bytes calldata params)
        external
        returns (bool)
    {
        require(msg.sender == address(lender), "!lender");
        require(initiator == address(this), "!me");
        uint256 id = uint256(bytes32(params));
        asset.approve(address(investor), amount);
        (address casset, bytes memory data) = investor.kill(id);
        (bytes32 strategy, address[] memory assets) = abi.decode(data, (bytes32, address[]));
        swap(casset);

        if (strategy == "sushi") {
            address a = assets[0];
            IERC20(a).transfer(a, IERC20(a).balanceOf(address(this)));
            ISushi(a).burn(address(this));
            swap(ISushi(a).token0());
            swap(ISushi(a).token1());
        }

        asset.approve(address(lender), amount + premium);
        asset.transfer(msg.sender, asset.balanceOf(address(this)) - amount - premium);
        return true;
    }

    /// @notice Internal function to swap assets using the Strategy Helper contract
    /// @param _fromAsset The address of the asset to be swapped
    function swap(address _fromAsset) internal {
        IERC20 fromAsset = IERC20(_fromAsset);
        uint256 amount = fromAsset.balanceOf(address(this));
        fromAsset.approve(address(sh), amount);
        sh.swap(_fromAsset, address(asset), amount, slippage, address(this));
    }
}
