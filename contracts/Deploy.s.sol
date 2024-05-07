// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

//import {Pool} from "./Pool.sol";
import {Bank} from "./Bank.sol";
import {Store} from "./Store.sol";
import {Investor} from "./Investor.sol";
import {PositionManager} from "./PositionManager.sol";
import {InvestorStrategyProxy} from "./InvestorStrategyProxy.sol";
import {Whitelist} from "./Whitelist.sol";
import {StrategySushiswap} from "./StrategySushiswap.sol";
import {StrategyCamelotV3} from "./StrategyCamelotV3.sol";
import {UtilFarmingBalances} from "./UtilFarmingBalances.sol";

interface Hevm {
    function startBroadcast() external;
    function stopBroadcast() external;
    function startPrank(address) external;
}

interface IWETH {
    function approve(address, uint256) external returns (bool);
}

interface IERC20 {
    function approve(address, uint256) external returns (bool);
}

interface Pool {
    function file(bytes32, address) external;
}

interface Multisig {
    function add(address target, uint256 value, bytes calldata data) external;
}


contract Deploy {
    Hevm vm = Hevm(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function run() external {
        vm.startBroadcast();

        salaries();
    }

    function stip() internal {
        address(0x08aa7480824f5B953A997d62a545382fE6071981).call(abi.encodeWithSignature("setMerkleRoot(uint256,bytes32)", 8, 0xa90ea1e0dd2c1ee7ad5cebb6866902002c8e69b6342e20607fc051e091a8f59e));
    }

    function debug() internal {
      vm.stopBroadcast();
      vm.startPrank(0x20dE070F1887f82fcE2bdCf5D6d9874091e6FAe9);
      address(0x768778aB1B2c4E462172136eE2584Ea7494bcB81).call(abi.encodeWithSignature(
        "open(uint256,address,uint256,uint256,address)",
        1, 0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8, 5000000, 10000000, 0x20dE070F1887f82fcE2bdCf5D6d9874091e6FAe9
      ));
    }

    function ms() internal {
        Multisig multisig = Multisig(0xaB7d6293CE715F12879B9fa7CBaBbFCE3BAc0A5a);
        multisig.add(0x49D6628C87F2f0E6A17edfD7E9DA95ED95a7d1B2, 0, abi.encodeWithSignature("setReserveRatio(uint256)", 8, 0xa90ea1e0dd2c1ee7ad5cebb6866902002c8e69b6342e20607fc051e091a8f59e));
    }

    function strategy() internal {
        address keeper = 0x3b1F14068Fa2AF4B08b578e80834bC031a52363D;
        address strategyHelper = 0x72f7101371201CeFd43Af026eEf1403652F115EE;
        address usdce = 0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8;
        address grail = 0x3d9907F9a368ad0a51Be60f7Da3b97cf940982D8;
        address isp = 0xA9737E08e8cF6cf7baE8096fd32A4E11E1159ffb;
        Investor i = Investor(0x780D46fef77ac5f83399BD2BE363125982A78973);
        StrategyCamelotV3 s = new StrategyCamelotV3(
          usdce, strategyHelper,
          0x3CAaE25Ee616f2C8E13C74dA0813402eae3F496b, // xGRAIL
          0x357Abb2d1Ef48D8F3afDA83D0515F75322d59DcA, // StrategyHelperUniswapV3
          0x1F1Ca4e8236CD13032653391dB7e9544a6ad123E, // Gamma UniProxy Camelot
          0x0Fc73040b26E9bC8514fA028D998E73A254Fa76E, // Camelot Quoter
          0xd7Ef5Ac7fd4AAA7994F3bc1D273eAb1d1013530E, // Gamma Hypervisor ETH/USDC
          0x3b6486154b9dAe942C393b1cB3d11E3395B02Df8, // Camelot NFTPool V3 ETH/USDC
          0xaf88d065e77c8cC2239327C5EDb3A432268e5831, // USDC
          abi.encodePacked(0xaf88d065e77c8cC2239327C5EDb3A432268e5831, 0x82aF49447D8a07e3bd95BD0d56f35241523fBab1)
        );
        s.file("exec", address(i));
        s.file("exec", address(isp));
        i.strategyUgrade(1, address(s));
        //i.strategyNew(1, address(s));
        //i.strategySetCap(1, 25_000e18);
        s.file("rewardToken1", grail);
        s.file("slippage", 150);
        s.file("keeper", keeper);

        IERC20(usdce).approve(address(s), 1e6);
        s.mint(1e6);
    }

    function whitelist() internal {
        address deployer = 0x20dE070F1887f82fcE2bdCf5D6d9874091e6FAe9;
        Investor i = Investor(0x780D46fef77ac5f83399BD2BE363125982A78973);
        PositionManager pm = PositionManager(0x768778aB1B2c4E462172136eE2584Ea7494bcB81);
        Whitelist w = new Whitelist();
        w.file("whitelist", deployer);
        i.file("whitelist", address(w));
        pm.file("whitelist", address(w));
        w.file("whitelist", address(pm));
    }

    function upgrade() internal {
        address oldi = 0x5cC6D18D4ca54Ff7e9a2aE2E0d45d8A04ABBAa8d;
        address strategyHelper = 0x72f7101371201CeFd43Af026eEf1403652F115EE;
        Bank p = Bank(0x34286e6BF46Fd4E68D95a7cF0C6ecf7180001a5D);
        Bank b = Bank(0xbC6519Ef876Fe8626602a399F9e1d8577610ffb4);
        Store s = Store(0xecC5E787E23c82099A52AD3f7C64Ccc68fc50cbA);
        InvestorStrategyProxy isp = InvestorStrategyProxy(0x2FeC87A1A73d83d8256080C3ca297658C08c83A6);
        PositionManager pm = PositionManager(0xA01EcDd5B099dbE9a2E890c4998EA68dBe8f44f2);

        Investor i = new Investor(address(s), strategyHelper);
        p.file("exec", address(i));
        b.file("exec", address(i));
        s.file("exec", address(i));
        isp.file("exec", address(i));
        i.file("strategyProxy", address(isp));
        pm.file("investor", address(i));
        p.file("exec", oldi);
        b.file("exec", oldi);
        s.file("exec", oldi);
        isp.file("exec", oldi);
        require(s.exec(address(i)) == true, "not added");
        require(p.exec(oldi) == false, "not removed");
        require(b.exec(oldi) == false, "not removed");
        require(s.exec(oldi) == false, "not removed");
        require(isp.exec(oldi) == false, "not removed");
    }

    function upgradeRevert() internal {
        Bank p = Bank(0x34286e6BF46Fd4E68D95a7cF0C6ecf7180001a5D);
        Bank b = Bank(0xbC6519Ef876Fe8626602a399F9e1d8577610ffb4);
        Store s = Store(0xecC5E787E23c82099A52AD3f7C64Ccc68fc50cbA);
        InvestorStrategyProxy isp = InvestorStrategyProxy(0x2FeC87A1A73d83d8256080C3ca297658C08c83A6);
        PositionManager pm = PositionManager(0xA01EcDd5B099dbE9a2E890c4998EA68dBe8f44f2);
    }

    function deploy() internal {
        address deployer = 0x20dE070F1887f82fcE2bdCf5D6d9874091e6FAe9;
        address keeper = 0x3b1F14068Fa2AF4B08b578e80834bC031a52363D;
        address strategyHelper = 0x72f7101371201CeFd43Af026eEf1403652F115EE;
        address weth = 0x82aF49447D8a07e3bd95BD0d56f35241523fBab1;
        address usdce = 0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8;

        /*
        Pool p = new Pool(
            usdce,
            0x627ea8812ef561D6ad9308D2A63e36901f51E3E5,
            0x50834F3163758fcC1Df9973b6e91f0F0F0434aD3,
            1e6,
            0.95e18,
            10_000e6
        );
        */
        Pool p = Pool(0x0032F5E1520a66C6E572e96A11fBF54aea26f9bE);
        //IERC20(usdce).approve(address(p), 100e6);
        //p.mint(100e6, deployer);

        InvestorStrategyProxy isp = new InvestorStrategyProxy(usdce);
        Investor i;
        {
            Store s = new Store();
            Bank b = new Bank();
            i = new Investor(address(s), strategyHelper);
            isp.file("exec", address(i));
            s.file("exec", address(i));
            s.setAddress(keccak256(abi.encode("POOL")), address(p));
            b.file("exec", address(i));
            //p.file("exec", address(i));
            i.file("bank", address(b));
            i.file("strategyProxy", address(isp));
            i.file("status", 4);
            i.collateralSetFactor(usdce, 0.95e18);
            i.collateralSetCap(usdce, 10_000e6);
            i.collateralSetFactor(weth, 0.9e18);
            i.collateralSetCap(weth, 10e18);
        }

        /*
        {
            StrategySushiswap sSushiSwapEthUsdc = new StrategySushiswap(usdce, strategyHelper, 0xF4d73326C13a4Fc5FD7A064217e12780e9Bd62c3, 0);
            i.strategyNew(1, address(sSushiSwapEthUsdc));
            i.strategySetCap(1, 10_000e18);
            sSushiSwapEthUsdc.file("exec", address(i));
            sSushiSwapEthUsdc.file("exec", address(isp));
            sSushiSwapEthUsdc.file("keeper", keeper);

            IERC20(usdce).approve(address(sSushiSwapEthUsdc), 1e6);
            sSushiSwapEthUsdc.mint(1e6);
        }
        */

        PositionManager pm = new PositionManager(address(i));
        //IERC20(usdce).approve(address(pm), 0.5e6);
        //pm.open(1, usdce, 0.5e6, 1e6, deployer);
    }

    function salaries() internal {
        Multisig multisig = Multisig(0xaB7d6293CE715F12879B9fa7CBaBbFCE3BAc0A5a);
        address rdo = 0x033f193b3Fceb22a440e89A2867E8FEE181594D9;
        address xrdo = 0x45a58482c3B8Ce0e8435E407fC7d34266f0A010D;
        address usdce = 0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8;
        address frank = 0xef93C8cb3318D782c70bD674e3D21636064F8ddE;
        address farmerc = 0xd6185a66F17Fce9016216931785eC5baf8F1D1e9;
        address jimbo = 0x055B29979f6BC27669Ebd54182588FeF12ffBFc0;
        address yieldl = 0x097F2358142F0C5fE544d8015E003ceD3C20f024;
        address jonas = 0x97A0E5872B47f31321fcaed57D293058bfA9C338;
        uint256 rdop = 0.06207e18;
        address[5] memory salariesAddress = [farmerc, frank, jimbo, yieldl, jonas];
        uint256[5] memory salariesUsdc = [uint256(0), 0, 0, 1500e6, 250e6];
        uint256[5] memory salariesRdo = [uint256(0), 0, 0, 0, 0];
        uint256[5] memory salariesXrdo = [uint256(22000e18), 14000e18, 11000e18, 750e18, 500e18];
        multisig.add(rdo, 0, abi.encodeWithSignature("approve(address,uint256)", xrdo, 48250e18*1e18/rdop));
        for (uint256 i = 0; i < salariesAddress.length; i++) {
            if (salariesUsdc[i] > 0) multisig.add(usdce, 0, abi.encodeWithSignature("transfer(address,uint256)", salariesAddress[i], salariesUsdc[i]));
            if (salariesRdo[i] > 0) multisig.add(rdo, 0, abi.encodeWithSignature("transfer(address,uint256)", salariesAddress[i], salariesRdo[i]*1e18/rdop));
            if (salariesXrdo[i] > 0) multisig.add(xrdo, 0, abi.encodeWithSignature("mint(uint256,address)", salariesXrdo[i]*1e18/rdop, salariesAddress[i]));
        }
    }

    function onERC721Received(address, address, uint256, bytes calldata) external pure returns (bytes4) {
        return Deploy.onERC721Received.selector;
    }
}

library console {
    address constant CONSOLE_ADDRESS = address(0x000000000000000000636F6e736F6c652e6c6f67);

    function _sendLogPayload(bytes memory payload) private view {
        uint256 payloadLength = payload.length;
        address consoleAddress = CONSOLE_ADDRESS;
        assembly {
            let payloadStart := add(payload, 32)
            let r := staticcall(gas(), consoleAddress, payloadStart, payloadLength, 0, 0)
        }
    }

    function log(uint p0) internal view {
        _sendLogPayload(abi.encodeWithSignature("log(uint)", p0));
    }

    function log(string memory p0) internal view {
        _sendLogPayload(abi.encodeWithSignature("log(string)", p0));
    }

    function log(address p0) internal view {
        _sendLogPayload(abi.encodeWithSignature("log(address)", p0));
    }

    function log(bytes32 p0) internal view {
        _sendLogPayload(abi.encodeWithSignature("log(bytes32)", p0));
    }

    function log(string memory p0, uint p1) internal view {
        _sendLogPayload(abi.encodeWithSignature("log(string,uint)", p0, p1));
    }
}
