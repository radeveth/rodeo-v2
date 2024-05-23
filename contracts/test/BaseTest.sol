// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Investor } from "../Investor.sol";
import { Store } from "../Store.sol";
import { Helper } from "../Helper.sol";
import { Whitelist } from "../Whitelist.sol";
import { InvestorStrategyProxy } from "../InvestorStrategyProxy.sol";
import { Bank } from "../Bank.sol";

// Interfaces
import { IPool } from "../interfaces/IPool.sol";
import { IStrategy } from "../interfaces/IStrategy.sol";
import { IOracle } from "../interfaces/IOracle.sol";

// Mocks
import { MockERC20 } from "./mock/MockERC20.sol";

contract BaseTest {

  Investor investor;
  Store store;
  Helper helper;
  Whitelist whitelist;
  InvestorStrategyProxy strategyProxy;
  Bank bank;
  IPool pool;
  IStrategy strategy;
  MockERC20 asset;
  IOracle oracle;

  address constant ADMIN = address(9999);
  address constant BOB = address(1);

  MockERC20 USDC;

  function prepareEnv() public {
    USDC = new MockERC20(); 
    USDC.initialize("USDC-Mock", "USDC", 18);
    
    store = new Store();
    // helper = new Helper();
  }

}