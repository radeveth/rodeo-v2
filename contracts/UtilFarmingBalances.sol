// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IInvestor {
    function nextPosition() external view returns (uint256);
    function positions(uint256) external view returns (address, address, uint256, uint256, uint256, uint256, uint256);
}

interface IPositionManager {
    function ownerOf(uint256) external view returns (address);
}

contract UtilFarmingBalances {
    function get(uint256 start) external view returns (address[] memory, uint256[] memory) {
        IInvestor i = IInvestor(0x8accf43Dd31DfCd4919cc7d65912A475BfA60369);
        IPositionManager pm = IPositionManager(0x5e4d7F61cC608485A2E4F105713D26D58a9D0cF6);
        uint256 max = i.nextPosition();
        address[] memory users = new address[](max-start);
        uint256[] memory balances = new uint256[](max-start);
        for (uint256 y = start; y < max; y++) {
            (,,,,,uint256 value,) = i.positions(y);
            (bool ok, bytes memory data) = address(pm).staticcall(abi.encodeWithSelector(IPositionManager.ownerOf.selector, y));
            if (ok) {
              users[y-start] = abi.decode(data, (address));
              balances[y-start] = value;
            }
        }
        return (users, balances);
    }
}
