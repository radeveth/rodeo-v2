package jobs

import (
	"app/lib"
	"app/models"
	"math/big"
	"net/url"
	"time"
)

var automationContracts = []string{
	"0x483F0f25B37f83ed45D94602E2483AA6151f04eD",
}

var _ = lib.RegisterSchedule("automations", time.Hour)

var _ = lib.RegisterJob("automations", func(c *lib.Ctx, args lib.J) {
	for _, contract := range automationContracts {
		client := c.Server.ChainClients[models.DefaultChainId]
		result := client.Call(contract, "canRun--bytes")
		if len(result[0].([]byte)) == 0 {
			lib.LogInfo("skipping", lib.J{"contract": contract})
			continue
		}
		lib.LogInfo("running", lib.J{"contract": contract})
		txHash := client.Call(contract, "+run-bytes-", []byte{})[0].(string)
		lib.LogInfo("ran", lib.J{"contract": contract, "txhash": txHash})
	}
})

var _ = lib.RegisterSchedule("automations-vaults", time.Hour)

var _ = lib.RegisterJob("automations-vaults", func(c *lib.Ctx, args lib.J) {
	//oneInchRouter := "0x1111111254EEB25477B68fb85Ed929f73A960582"
	wethAddress := "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1"
	for _, v := range models.Vaults {
		client := c.Server.ChainClients[models.DefaultChainId]
		result := client.Call(v.Strategy, "info--int256,uint256,uint256,uint256,uint256")
		leverage := lib.Bni(result[1])
		assets := lib.Bni(result[2])
		balance := lib.Bni(result[3])
		debt := lib.Bni(result[4])
		result = client.Call(v.Asset, "balanceOf-address-uint256", v.Address)
		assetsInVault := lib.Bni(result[0])
		result = client.Call("0xb523AE262D20A936BC152e6023996e46FDC2A95D", "latestAnswer--int256")
		wstEthToEthRate := lib.Bni(result[0])
		deposits := assets.Add(assetsInVault)
		reserve := deposits.Mul(lib.Bn(10, 0)).Div(lib.Bn(100, 0))
		targetBorrow := deposits.Sub(reserve).Mul(v.TargetLeverage.Sub(lib.ONE)).Div(lib.ONE)
		if leverage.Gt(v.TargetLeverage.Mul(lib.Bn(120, 0)).Div(lib.Bn(100, 0))) {
			borrowChange := debt.Sub(targetBorrow.Mul(wstEthToEthRate).Div(lib.ONE))
			result = client.Call(wethAddress, "balanceOf-address-uint256", v.Strategy)
			wethInStrategy := lib.Bni(result[0])

			targetBalance := deposits.Mul(v.TargetLeverage).Div(lib.ONE).Sub(reserve)
			wstethWithdraw := balance.Sub(targetBalance)
			swapAmount := borrowChange.Mul(lib.ONE).Div(wstEthToEthRate).Mul(lib.Bns("103")).Div(lib.Bns("100"))
			swapTo, swapData := oneInchQuote(
				v.Asset,
				wethAddress,
				v.Strategy,
				swapAmount.String(),
			)
			lib.LogInfo("deleveraging", lib.J{
				"vault":    v.Address,
				"withdraw": wstethWithdraw.String(),
				"repay":    borrowChange.String(),
			})
			txHash := client.Call(v.Strategy,
				"+act-uint256,uint256,uint256,uint256,address,bytes-",
				big.NewInt(2), swapAmount, borrowChange.Sub(wethInStrategy), swapAmount, swapTo, swapData)[0].(string)
			lib.LogInfo("deleveraged", lib.J{
				"vault":    v.Address,
				"withdraw": wstethWithdraw.String(),
				"repay":    borrowChange.String(),
				"txhash":   txHash,
			})
		} else if leverage.Lt(v.TargetLeverage.Mul(lib.Bn(80, 0)).Div(lib.Bn(100, 0))) {
			takeFromVault := assetsInVault.Sub(reserve)
			borrow := targetBorrow.Mul(wstEthToEthRate).Div(lib.ONE).Sub(debt)
			swapTo, swapData := oneInchQuote(
				wethAddress,
				v.Asset,
				v.Strategy,
				borrow.String(),
			)
			lib.LogInfo("leveraging", lib.J{
				"vault":  v.Address,
				"borrow": takeFromVault.String(),
				"debt":   borrow.String(),
			})
			txHash := client.Call(v.Strategy,
				"+act-uint256,uint256,uint256,uint256,address,bytes-",
				big.NewInt(1), takeFromVault, borrow, borrow, swapTo, swapData)[0].(string)
			lib.LogInfo("leveraged", lib.J{
				"vault":  v.Address,
				"borrow": takeFromVault.String(),
				"debt":   borrow.String(),
				"txhash": txHash,
			})
		}
	}
})

func oneInchQuote(from, to, caller, amount string) (string, string) {
	swap := struct {
		Tx struct {
			To   string
			Data string
		}
	}{}
	query := url.Values{}
	query.Set("src", from)
	query.Set("dst", to)
	query.Set("from", caller)
	query.Set("amount", amount)
	query.Set("slippage", "2")
	query.Set("disableEstimate", "true")
	lib.GetJSON("https://api.1inch.dev/swap/v6.0/42161/swap?"+query.Encode(), &swap, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + lib.Env("ONEINCH_API_KEY", ""),
	})
	return swap.Tx.To, swap.Tx.Data
}
