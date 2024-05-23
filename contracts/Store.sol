// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IStore } from "./interfaces/IStore.sol";

/**
 * @title Store
 * @dev Storage contract for managing different types of data.
 */
contract Store is IStore {
    mapping(address => bool) public exec;
    mapping(bytes32 => uint256) public uintValues;
    mapping(bytes32 => int256) public intValues;
    mapping(bytes32 => address) public addressValues;
    mapping(bytes32 => bool) public boolValues;
    mapping(bytes32 => string) public stringValues;
    mapping(bytes32 => bytes32) public bytes32Values;
    mapping(bytes32 => bytes32[]) public bytes32ArrayValues;
    mapping(bytes32 => Bytes32Set) internal bytes32Sets;

    struct Bytes32Set {
        bytes32[] values;
        mapping(bytes32 => uint256) positions;
    }

    event File(bytes32 indexed what, address data);

    error InvalidFile();
    error Unauthorized();
    error OverOrUnderflow();

    /**
     * @dev Initializes the contract by setting the deployer as an executor.
     */
    constructor() {
        exec[msg.sender] = true;
    }

    /**
     * @dev Modifier to check if the caller is authorized.
     */
    modifier auth() {
        if (!exec[msg.sender]) revert Unauthorized();
        _;
    }

    /**
     * @notice Updates contract configuration.
     * @param what The configuration key.
     * @param data The address value to be set.
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
     * @notice Retrieves a stored unsigned integer value.
     * @param key The key associated with the value.
     * @return The stored unsigned integer value.
     */
    function getUint(bytes32 key) external view returns (uint256) {
        return uintValues[key];
    }

    /**
     * @notice Sets a stored unsigned integer value.
     * @param key The key associated with the value.
     * @param value The unsigned integer value to be stored.
     * @return The stored unsigned integer value.
     */
    function setUint(bytes32 key, uint256 value) external auth returns (uint256) {
        uintValues[key] = value;
        return value;
    }

    /**
     * @notice Adjusts a stored unsigned integer value by a delta.
     * @param key The key associated with the value.
     * @param value The delta to be applied to the stored value.
     * @return The adjusted unsigned integer value.
     */
    function setUintDelta(bytes32 key, int256 value) external auth returns (uint256) {
        uint256 prev = uintValues[key];
        uint256 next = uint256(int256(prev) + value);
        if (value > 0 && next < prev) revert OverOrUnderflow();
        if (value < 0 && next > prev) revert OverOrUnderflow();
        uintValues[key] = next;
        return next;
    }

    /**
     * @notice Removes a stored unsigned integer value.
     * @param key The key associated with the value.
     */
    function removeUint(bytes32 key) external auth {
        delete uintValues[key];
    }

    /**
     * @notice Retrieves a stored signed integer value.
     * @param key The key associated with the value.
     * @return The stored signed integer value.
     */
    function getInt(bytes32 key) external view returns (int256) {
        return intValues[key];
    }

    /**
     * @notice Sets a stored signed integer value.
     * @param key The key associated with the value.
     * @param value The signed integer value to be stored.
     * @return The stored signed integer value.
     */
    function setInt(bytes32 key, int256 value) external auth returns (int256) {
        intValues[key] = value;
        return value;
    }

    /**
     * @notice Adjusts a stored signed integer value by a delta.
     * @param key The key associated with the value.
     * @param value The delta to be applied to the stored value.
     * @return The adjusted signed integer value.
     */
    function setIntDelta(bytes32 key, int256 value) external auth returns (int256) {
        int256 next = intValues[key] + value;
        intValues[key] = next;
        return next;
    }

    /**
     * @notice Removes a stored signed integer value.
     * @param key The key associated with the value.
     */
    function removeInt(bytes32 key) external auth {
        delete intValues[key];
    }

    /**
     * @notice Retrieves a stored address value.
     * @param key The key associated with the value.
     * @return The stored address value.
     */
    function getAddress(bytes32 key) external view returns (address) {
        return addressValues[key];
    }

    /**
     * @notice Sets a stored address value.
     * @param key The key associated with the value.
     * @param value The address value to be stored.
     * @return The stored address value.
     */
    function setAddress(bytes32 key, address value) external auth returns (address) {
        addressValues[key] = value;
        return value;
    }

    /**
     * @notice Removes a stored address value.
     * @param key The key associated with the value.
     */
    function removeAddress(bytes32 key) external auth {
        delete addressValues[key];
    }

    /**
     * @notice Retrieves a stored boolean value.
     * @param key The key associated with the value.
     * @return The stored boolean value.
     */
    function getBool(bytes32 key) external view returns (bool) {
        return boolValues[key];
    }

    /**
     * @notice Sets a stored boolean value.
     * @param key The key associated with the value.
     * @param value The boolean value to be stored.
     * @return The stored boolean value.
     */
    function setBool(bytes32 key, bool value) external auth returns (bool) {
        boolValues[key] = value;
        return value;
    }

    /**
     * @notice Removes a stored boolean value.
     * @param key The key associated with the value.
     */
    function removeBool(bytes32 key) external auth {
        delete boolValues[key];
    }

    /**
     * @notice Retrieves a stored string value.
     * @param key The key associated with the value.
     * @return The stored string value.
     */
    function getString(bytes32 key) external view returns (string memory) {
        return stringValues[key];
    }

    /**
     * @notice Sets a stored string value.
     * @param key The key associated with the value.
     * @param value The string value to be stored.
     * @return The stored string value.
     */
    function setString(bytes32 key, string memory value) external auth returns (string memory) {
        stringValues[key] = value;
        return value;
    }

    /**
     * @notice Removes a stored string value.
     * @param key The key associated with the value.
     */
    function removeString(bytes32 key) external auth {
        delete stringValues[key];
    }

    /**
     * @notice Retrieves a stored bytes32 value.
     * @param key The key associated with the value.
     * @return The stored bytes32 value.
     */
    function getBytes32(bytes32 key) external view returns (bytes32) {
        return bytes32Values[key];
    }

    /**
     * @notice Sets a stored bytes32 value.
     * @param key The key associated with the value.
     * @param value The bytes32 value to be stored.
     * @return The stored bytes32 value.
     */
    function setBytes32(bytes32 key, bytes32 value) external auth returns (bytes32) {
        bytes32Values[key] = value;
        return value;
    }

    /**
     * @notice Removes a stored bytes32 value.
     * @param key The key associated with the value.
     */
    function removeBytes32(bytes32 key) external auth {
        delete bytes32Values[key];
    }

    /**
     * @notice Retrieves a stored bytes32 array value.
     * @param key The key associated with the value.
     * @return The stored bytes32 array value.
     */
    function getBytes32Array(bytes32 key) external view returns (bytes32[] memory) {
        return bytes32ArrayValues[key];
    }

    /**
     * @notice Sets a stored bytes32 array value.
     * @param key The key associated with the value.
     * @param value The bytes32 array value to be stored.
     */
    function setBytes32Array(bytes32 key, bytes32[] memory value) external auth {
        bytes32ArrayValues[key] = value;
    }

    /**
     * @notice Removes a stored bytes32 array value.
     * @param key The key associated with the value.
     */
    function removeBytes32Array(bytes32 key) external auth {
        delete bytes32ArrayValues[key];
    }

    /**
     * @notice Checks if a bytes32 value exists in the set.
     * @param key The key associated with the set.
     * @param value The bytes32 value to check.
     * @return True if the value exists, false otherwise.
     */
    function containsBytes32(bytes32 key, bytes32 value) external view returns (bool) {
        return bytes32Sets[key].positions[value] != 0;
    }

    /**
     * @notice Retrieves the count of bytes32 values in the set.
     * @param key The key associated with the set.
     * @return The count of bytes32 values in the set.
     */
    function getBytes32Count(bytes32 key) external view returns (uint256) {
        return bytes32Sets[key].values.length;
    }

    /**
     * @notice Retrieves a range of bytes32 values from the set.
     * @param key The key associated with the set.
     * @param start The starting index.
     * @param end The ending index.
     * @return v The range of bytes32 values.
     */
    function getBytes32ValuesAt(bytes32 key, uint256 start, uint256 end) external view returns (bytes32[] memory v) {
        v = new bytes32[](end - start);
        for (uint256 i = start; i < end; i++) {
            v[i - start] = bytes32Sets[key].values[i];
        }
    }

    /**
     * @notice Adds a bytes32 value to the set.
     * @param key The key associated with the set.
     * @param value The bytes32 value to add.
     */
    function addBytes32(bytes32 key, bytes32 value) external auth {
        Bytes32Set storage s = bytes32Sets[key];
        if (s.positions[value] == 0) {
            s.values.push(value);
            s.positions[value] = s.values.length;
        }
    }

    /**
     * @notice Removes a bytes32 value from the set.
     * @param key The key associated with the set.
     * @param value The bytes32 value to remove.
     */
    function removeBytes32(bytes32 key, bytes32 value) external auth {
        Bytes32Set storage s = bytes32Sets[key];
        uint256 position = s.positions[value];
        if (position != 0) {
            uint256 valueIndex = position - 1;
            uint256 lastIndex = s.values.length - 1;
            if (valueIndex != lastIndex) {
                bytes32 lastValue = s.values[lastIndex];
                s.values[valueIndex] = lastValue;
                s.positions[lastValue] = position;
            }
            s.values.pop();
            delete s.positions[value];
        }
    }
}
