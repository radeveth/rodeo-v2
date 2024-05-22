// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title IInvestor
 * @notice Interface for the Investor contract to get position details
 */
interface IInvestor {
    /**
     * @notice Get the next position ID
     * @return The next position ID
     */
    function nextPosition() external view returns (uint256);

    /**
     * @notice Get the details of a position
     * @param positionId The ID of the position
     * @return The details of the position
     */
    function positions(uint256 positionId)
        external
        view
        returns (address, address, uint256, uint256, uint256, uint256, uint256);
}

/**
 * @title IPositionManager
 * @notice Interface for the Position Manager contract to get the owner of a position
 */
interface IPositionManager {
    /**
     * @notice Get the owner of a position
     * @param positionId The ID of the position
     * @return The address of the owner of the position
     */
    function ownerOf(uint256 positionId) external view returns (address);
}

/**
 * @title UtilFarmingBalances
 * @notice Utility contract to get farming balances and associated users
 */
contract UtilFarmingBalances {
    /**
     * @notice Get the users and their farming balances starting from a given position
     * @param start The starting position ID
     * @return users An array of user addresses
     * @return balances An array of user balances
     */
    function get(uint256 start) external view returns (address[] memory, uint256[] memory) {
        IInvestor i = IInvestor(0x8accf43Dd31DfCd4919cc7d65912A475BfA60369);
        IPositionManager pm = IPositionManager(0x5e4d7F61cC608485A2E4F105713D26D58a9D0cF6);
        uint256 max = i.nextPosition();
        address[] memory users = new address[](max - start);
        uint256[] memory balances = new uint256[](max - start);
        for (uint256 y = start; y < max; y++) {
            (,,,,, uint256 value,) = i.positions(y);
            (bool ok, bytes memory data) =
                address(pm).staticcall(abi.encodeWithSelector(IPositionManager.ownerOf.selector, y));
            if (ok) {
                users[y - start] = abi.decode(data, (address));
                balances[y - start] = value;
            }
        }
        return (users, balances);
    }
}
