import { createWeb3Modal } from "@web3modal/wagmi";
import { arbitrum } from "@wagmi/core/chains";
import { coinbaseWallet, walletConnect, injected } from "@wagmi/connectors";
import {
  parseAbi,
  parseUnits,
  formatUnits,
  publicActions,
  walletActions,
} from "viem";
import {
  connect,
  reconnect,
  disconnect,
  http,
  createConfig,
  getAccount,
  watchAccount,
  getConnectorClient,
} from "@wagmi/core";

window.chain =
  window.chain ||
  (function () {
    const ONE = 1_000_000_000_000_000_000n;
    const ONE12 = 1_000_000_000_000n;
    const UINT_MAX =
      115792089237316195423570985008687907853269984665640564039457584007913129639935n;
    const ZERO_ADDRESS = "0x0000000000000000000000000000000000000000";
    const projectId = window.env.WC_PROJECT_ID;
    const defaultChain = arbitrum;
    const url = window.location.protocol + "://" + window.location.host + "/";
    const metadata = {
      name: window.env.COMPANY_NAME,
      description: "",
      url: url,
      icons: [url + "favicon.png"],
    };
    const config = createConfig({
      chains: [arbitrum],
      transports: {
        [arbitrum.id]: http(),
      },
      connectors: [
        walletConnect({ projectId, metadata, showQrModal: false }),
        injected({ shimDisconnect: true }),
        coinbaseWallet({
          appName: metadata.name,
          appLogoUrl: metadata.icons[0],
        }),
      ],
    });
    const modal = createWeb3Modal({ wagmiConfig: config, projectId });
    reconnect(config);
    watchAccount(config, {
      onChange: (account) => {
        const address = account.address ?? "";
        if (getCookie("address") !== address) {
          setCookie("address", address);
          window.location.reload();
        }
      },
    });

    document.addEventListener("turbo:load", () => {
      // Highlight current link
      document.querySelectorAll("a").forEach((el) => {
        if (window.location.href.startsWith(el.href)) {
          el.classList.add("current");
        }
      });

      // Show last tx hash notification
      const lastTxHash = getCookie("txhash");
      if (lastTxHash) {
        console.log("tx hash", lastTxHash);
        txhashOverlay.innerText =
          "Confirmed: " + lastTxHash.slice(0, 20) + "...";
        txhashOverlay.href = "https://arbiscan.io/tx/" + lastTxHash;
        txhashOverlay.style.display = "block";
        setTimeout(() => (txhashOverlay.style.display = "none"), 10000);
        setCookie("txhash", "");
      }
    });

    function getCookie(name) {
      var v = document.cookie.match("(^|;) ?" + name + "=([^;]*)(;|$)");
      return v ? v[2] : null;
    }

    function setCookie(name, value) {
      var d = new Date();
      d.setTime(d.getTime() + 24 * 60 * 60 * 1000 * 365);
      document.cookie =
        name + "=" + value + ";path=/;expires=" + d.toGMTString();
    }

    function formatNumber(amount, decimals = 18, decimalsShown = 2) {
      amount = parseFloat(formatUnits(amount, decimals));
      return Intl.NumberFormat("en-US", {
        useGrouping: true,
        minimumFractionDigits: decimalsShown,
        maximumFractionDigits: decimalsShown,
      }).format(amount);
    }

    function formatError(e) {
      return e.message;
    }

    async function call(address, fn, ...args) {
      const client = (await getConnectorClient(config))
        .extend(publicActions)
        .extend(walletActions);
      let [name, params, returns] = fn.split("-");
      let rname = name[0] === "+" ? name.slice(1) : name;
      if (rname[0] === "$") rname = rname.slice(1);
      let efn = `function ${rname}(${params}) external`;
      if (name[0] !== "+") efn += " view";
      if (returns) efn += ` returns (${returns})`;
      const isPayable = name[1] === "$";
      const abi = parseAbi([efn]);
      const { request } = await client.simulateContract({
        address,
        abi,
        value: isPayable ? args[0] : 0n,
        args: isPayable ? args.slice(1) : args,
        functionName: rname,
      });
      request.chain = defaultChain;
      const hash = await client.writeContract(request);
      await client.waitForTransactionReceipt({ hash });
      console.log("tx hash", hash);
      return hash;
    }

    async function runTransaction(fn) {
      try {
        error.style.display = "none";
        loadingOverlay.style.display = "block";
        const hash = await fn();
        if (hash) setCookie("txhash", hash);
        Turbo.visit(window.location.pathname, { action: "replace" });
      } catch (e) {
        console.log(e.error, e.data, Object.entries(e));
        console.error(e);
        error.innerText = formatError(e);
        error.style.display = "block";
      } finally {
        loadingOverlay.style.display = "none";
      }
    }

    function getAddress() {
      return getAccount(config).address;
    }

    function onConnect() {
      //console.log(getAccount(config));
      if (getAccount(config).isConnected) {
        modal.open();
        //disconnect(config);
      } else {
        modal.open();
        //reconnect(config);
      }
    }

    function onEarn(e, pool, token, tab, shares, supplied, allowance) {
      e.preventDefault();
      let parsedAmount;
      let allowanceOk = false;
      try {
        parsedAmount = parseUnits(amount.value, 6);
        allowanceOk = parseUnits(allowance, 0) >= parsedAmount;
      } catch (e) {
        error.innerText = "Invalid amount";
        error.style.display = "block";
        return;
      }
      if (tab == "deposit") {
        runTransaction(async () => {
          if (!allowanceOk) {
            await call(token, "+approve-address,uint256-", pool, UINT_MAX);
          }
          return await call(
            pool,
            "+mint-uint256,address-",
            parsedAmount,
            getAddress()
          );
        });
      } else {
        let burnShares = (parsedAmount * BigInt(shares)) / BigInt(supplied);
        if (burnShares > BigInt(shares)) {
          burnShares = BigInt(shares);
        }
        runTransaction(() => {
          return call(pool, "+burn-uint256,address-", burnShares, getAddress());
        });
      }
    }

    function onSilos(e, rdo, xrdo, tokenStakingDividends, tab, allowance) {
      e.preventDefault();
      let parsedAmount;
      let allowanceOk = false;
      try {
        parsedAmount = parseUnits(window.amount ? amount.value : "0", 18);
        allowanceOk = parseUnits(allowance, 0) >= parsedAmount;
      } catch (e) {
        console.error(e);
        error.innerText = "Invalid amount";
        error.style.display = "block";
        return;
      }
      if (tab == "deposit") {
        runTransaction(async () => {
          if (!allowanceOk) {
            await call(rdo, "+approve-address,uint256-", xrdo, UINT_MAX);
          }
          return await call(
            xrdo,
            "+mintAndAllocate-uint256,uint256,address-",
            0,
            parsedAmount,
            getAddress()
          );
        });
      } else if (tab == "withdraw") {
        runTransaction(async () => {
          await call(xrdo, "+deallocate-uint256,uint256-", 0, parsedAmount);
          return await call(
            xrdo,
            "+burn-uint256,uint256-",
            (parsedAmount * 99n) / 100n,
            parseInt(amountTime.value) * 86400
          );
        });
      } else if (tab == "claim") {
        runTransaction(async () => {
          return await call(tokenStakingDividends, "+claim--");
        });
      }
    }

    function updateSilo() {
      if (!window.amountTime) return;
      const time = parseInt(amountTime.value);
      vestingLabel.innerText = `Vesting time (${time} days, ${(
        time / 30
      ).toFixed(1)} months)`;
      vestingEnds.innerText = new Date(Date.now() + time * 86400 * 1000)
        .toISOString()
        .split("T")[0];
      try {
        const parsedAmount = parseUnits(amount.value, 6);
        vestingTokens.innerText =
          (parsedAmount *
            (ONE / 2n + ((ONE / 2n) * (BigInt(time) - 15n)) / 165n)) /
          ONE /
          1000000n;
        vestingTokens.innerText += " RDO";
      } catch (e) {
        console.error(e);
      }
    }

    function onRewards(e, stipDistributor, week, amount, proof) {
      e.preventDefault();
      runTransaction(async () => {
        return await call(
          stipDistributor,
          "+claim-address,uint256[],uint256[],bytes32[][]--",
          getAddress(),
          [week],
          [amount],
          [JSON.parse(proof)]
        );
      });
    }

    function onVesting(e, vester, index) {
      e.preventDefault();
      runTransaction(async () => {
        return await call(
          vester,
          "+claim-uint256,address--",
          index,
          ZERO_ADDRESS
        );
      });
    }

    function updateFarmOpen(earnApy, farmApy, price) {
      earnApy = parseUnits(earnApy, 0);
      farmApy = parseUnits(farmApy, 0);
      price = parseUnits(price, 0);
      const borrow = parseUnits(amountBorrow.value, 18);
      const collateral = parseUnits(amountCollateral.value, 18);
      console.log(earnApy, farmApy, price, borrow, collateral);
      const leverage = (borrow * ONE) / ((collateral * price) / ONE);
      console.log(leverage);
      numberLeverage.innerText = formatNumber(leverage, 18, 1);
      numberFarm.innerText = formatNumber((leverage * farmApy) / ONE, 16, 1);
      numberEarn.innerText = formatNumber((leverage * earnApy) / ONE, 16, 1);
      numberNet.innerText = formatNumber(
        (leverage * (farmApy - earnApy)) / ONE,
        16,
        1
      );
    }

    async function onFarmOpen(
      e,
      positionManager,
      strategy,
      token,
      tokenDecimals,
      allowance
    ) {
      e.preventDefault();
      let collateral;
      let borrow;
      let allowanceOk = false;
      try {
        collateral = parseUnits(amountCollateral.value, tokenDecimals);
        borrow = parseUnits(amountBorrow.value, 6);
        allowanceOk = parseUnits(allowance, 0) >= collateral;
      } catch (e) {
        error.innerText = "Invalid amount";
        error.style.display = "block";
        return;
      }
      await runTransaction(async () => {
        if (!allowanceOk) {
          await call(
            token,
            "+approve-address,uint256-",
            positionManager,
            UINT_MAX
          );
        }
        return await call(
          positionManager,
          "+open-uint256,address,uint256,uint256,address-",
          strategy,
          token,
          collateral,
          borrow,
          getAddress()
        );
      });
      //Turbo.visit("/farm/", { action: "replace" });
    }

    function updateFarmEdit(position, tokenPrice, tokenDecimals, apy) {
      position = JSON.parse(position);
      console.log(position, tokenPrice);
      const collateralValue =
        (parseUnits(position.collateral, 0) * parseUnits(tokenPrice, 0)) /
        parseUnits("1", tokenDecimals);
      const leverage =
        (parseUnits(position.shares_value, 0) * ONE) / collateralValue;
      numberLeverage.innerText = formatNumber(leverage, 18, 1) + "x";
      numberApy.innerText = formatNumber(parseUnits(apy, 0), 16, 1) + "%";
      numberValue.innerText = formatNumber(
        parseUnits(position.shares_value, 0),
        18,
        2
      );
      numberBorrow.innerText = formatNumber(
        parseUnits(position.borrow_value, 0),
        18,
        2
      );
      numberCollateral.innerText = formatNumber(
        (parseUnits(position.collateral, 0) * parseUnits(tokenPrice, 0)) / ONE,
        tokenDecimals,
        2
      );
      numberCollateralToken.innerText = formatNumber(
        parseUnits(position.collateral, 0),
        tokenDecimals,
        2
      );
    }

    async function onFarmEdit(
      e,
      positionManager,
      tab,
      position,
      tokenDecimals,
      allowance
    ) {
      e.preventDefault();
      position = JSON.parse(position);
      let parsedAmount;
      let allowanceOk = true;
      let decimals = tab == "borrow" ? 6 : tab == "repay" ? 18 : tokenDecimals;
      try {
        parsedAmount = parseUnits(amount.value, decimals);
        if (tab == "deposit") {
          allowanceOk = parseUnits(allowance, 0) >= parsedAmount;
        }
      } catch (e) {
        error.innerText = "Invalid amount";
        error.style.display = "block";
        return;
      }
      await runTransaction(async () => {
        if (!allowanceOk) {
          await call(
            position.token,
            "+approve-address,uint256-",
            positionManager,
            UINT_MAX
          );
        }
        if (tab == "borrow") {
          return await call(
            positionManager,
            "+edit-uint256,int256,int256-",
            position.index,
            parsedAmount,
            0n
          );
        } else if (tab == "repay") {
          let shares =
            (parsedAmount * parseUnits(position.shares, 0)) /
            parseUnits(position.shares_value, 0);
          if (shares > parseUnits(position.shares, 0)) {
            shares = parseUnits(position.shares, 0);
          }
          return await call(
            positionManager,
            "+edit-uint256,int256,int256-",
            position.index,
            0n - shares,
            0n
          );
        } else if (tab == "deposit") {
          return await call(
            positionManager,
            "+edit-uint256,int256,int256-",
            position.index,
            0n,
            parsedAmount
          );
        } else if (tab == "withdraw") {
          return await call(
            positionManager,
            "+edit-uint256,int256,int256-",
            position.index,
            0n,
            0n - parsedAmount
          );
        }
      });
    }

    return {
      call,
      getCookie,
      setCookie,
      getAddress,
      onConnect,
      onEarn,
      onSilos,
      updateSilo,
      onRewards,
      onVesting,
      onFarmOpen,
      updateFarmOpen,
      onFarmEdit,
      updateFarmEdit,
    };
  })();
