package controllers

import (
	"app/lib"
	"app/models"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var _ = lib.RegisterSchedule("cache-prime-1m", time.Hour)
var _ = lib.RegisterJob("cache-prime-1m", func(c *lib.Ctx, args lib.J) {
	c.Cache.Set("pool", cachePool(c)(), 75*time.Second)
})
var _ = lib.RegisterSchedule("cache-prime-5m", time.Hour)
var _ = lib.RegisterJob("cache-prime-5m", func(c *lib.Ctx, args lib.J) {
	c.Cache.Set("strategies", cacheStrategies(c)(), 6*time.Minute)
	c.Cache.Set("collaterals", cacheCollaterals(c)(), 6*time.Minute)
})

func cacheStrategies(c *lib.Ctx) func() interface{} {
	return func() interface{} {
		client := c.Server.ChainClients[models.DefaultChainId]
		poolInfo := client.Call(models.AddressHelper, "pool-address-bool,uint256,uint256,uint256,uint256,uint256,uint256,uint256,uint256", models.Pools[0].Address)
		borrowApy := lib.Bni(poolInfo[7]).Mul(lib.YEAR)
		strategies := models.Strategies
		defillamaPools := struct {
			Data []struct {
				Pool   string
				Apy    float64
				TvlUsd float64
			}
		}{}
		lib.GetJSON("https://yields.llama.fi/pools", &defillamaPools, nil)
		for _, s := range strategies {
			result := client.Call(models.AddressInvestor, "getStrategy-uint256-address,uint256,uint256", big.NewInt(s.Index))
			s.Address = result[0].(common.Address).String()
			s.Cap = lib.Bni(result[1])
			s.Status = lib.Bni(result[2]).Std().Int64()
			totalShares := lib.Bni(client.Call(s.Address, "totalShares--uint256")[0])
			s.Tvl = lib.Bni(client.Call(s.Address, "rate-uint256-uint256", totalShares)[0])
			if s.ApyType == "defillama" {
				for _, p := range defillamaPools.Data {
					if s.ApyId == p.Pool {
						s.Apy = lib.Bnf(p.Apy, 16)
						s.TvlTotal = lib.Bnf(p.TvlUsd, 18)
						break
					}
				}
			}
			s.ApyWithLeverage = s.Apy.Sub(borrowApy).Mul(lib.Bn(5, 0))
		}
		return strategies
	}
}

func cacheCollaterals(c *lib.Ctx) func() interface{} {
	return func() interface{} {
		client := c.Server.ChainClients[models.DefaultChainId]
		collaterals := []*models.Collateral{}
		for _, t := range models.Collaterals {
			d := client.Call(t.Oracle, "decimals--uint8")[0].(uint8)
			price := lib.Bni(client.Call(t.Oracle, "latestAnswer--int256")[0]).Mul(lib.ONE).Div(lib.Bn(1, int64(d)))
			balance := lib.Bni(client.Call(t.Address, "balanceOf-address-uint256", models.AddressBank)[0])
			key0 := crypto.Keccak256(common.LeftPadBytes(common.HexToAddress(t.Address).Bytes(), 32), common.FromHex("0xb3bc6d089762efe6c36fe824d78650881aa025aca8a39468c3c09ce8509152e0"))
			var key [32]byte
			copy(key[:], key0[:32])
			cap := lib.Bni(client.Call(models.AddressStore, "getUint-bytes32-uint256", key)[0])
			collaterals = append(collaterals, &models.Collateral{
				Token:           t,
				Price:           price,
				Balance:         balance,
				UserBalance:     lib.ZERO,
				PositionBalance: lib.ZERO,
				Cap:             cap,
			})
		}
		return collaterals
	}
}

func cachePool(c *lib.Ctx) func() interface{} {
	return func() interface{} {
		client := c.Server.ChainClients[models.DefaultChainId]
		pool := models.Pools[0]
		result := client.Call(models.AddressHelper, "pool-address-bool,uint256,uint256,uint256,uint256,uint256,uint256,uint256,uint256", pool.Address)
		pool.Paused = result[0].(bool)
		pool.BorrowMin = lib.Bni(result[1])
		pool.Cap = lib.Bni(result[2])
		pool.Index = lib.Bni(result[3])
		pool.Shares = lib.Bni(result[4])
		pool.Borrow = lib.Bni(result[5])
		pool.Supply = lib.Bni(result[6])
		pool.Rate = lib.Bni(result[7]).Mul(lib.Bnf(365*24*60*60, 0))
		pool.Price = lib.Bni(result[8])
		pool.SupplyRate = lib.ZERO
		pool.Utilisation = lib.ZERO
		if pool.Supply.Gt(lib.ZERO) {
			pool.Utilisation = pool.Borrow.Mul(lib.ONE).Div(pool.Supply)
			pool.SupplyRate = pool.Rate.Mul(pool.Utilisation).Div(lib.ONE)
		}
		arbPrice := client.CallUint("0x3AC1bfc26e8c3FcF55e8E41DBb7910b38Da62eBD", "latestAnswer--int256")
		pool.ArbRate = lib.Bn(250_000, 10).Mul(arbPrice).Mul(lib.Bn(26, 18)).Div(pool.Supply.Add(pool.Borrow).Mul(lib.ONE12))
		return pool
	}
}

func AppStrategies(c *lib.Ctx) {
	client := c.Server.ChainClients[models.DefaultChainId]
	pool := &models.PoolInfo{}
	c.Cache.Try("pool", pool, time.Minute, cachePool(c))
	strategies := models.Strategies
	c.Cache.Try("strategies", &strategies, 5*time.Minute, cacheStrategies(c))
	collaterals := []*models.Collateral{}
	c.Cache.Try("collaterals", &collaterals, 5*time.Minute, cacheCollaterals(c))

	tab := c.Param("tab", "")
	index := lib.StringToInt(c.Param("s", "0"))
	var strategy *models.StrategyInfo
	for _, s := range strategies {
		if s.Index == index {
			strategy = s
			break
		}
	}

	tokenPrices := map[string]*lib.BigInt{}
	for _, c := range collaterals {
		tokenPrices[c.Token.Address] = c.Price
	}

	// Strategie's positions
	positions := []*models.Position{}
	positionLeverages := map[int64]*lib.BigInt{}
	positionApys := map[int64]*lib.BigInt{}
	positionMaxBorrows := map[int64]*lib.BigInt{}
	whitelisted := false
	if address := c.GetCookie("address"); address != "" {
		positionCount := lib.Bni(client.Call(models.AddressPositionManager, "balanceOf-address-uint256", address)[0])
		positionIds := client.Call(models.AddressPositionManager, "tokensOfOwner-address,uint256,uint256-uint256[]", address, big.NewInt(0), positionCount.Std())[0].([]*big.Int)
		for _, id := range positionIds {
			position := client.Call(models.AddressInvestor, "getPosition-uint256-address,uint256,uint256,address,uint256,uint256,uint256,uint256", id)
			if lib.Bni(position[4]).Eq(lib.ZERO) {
				continue
			}
			strategyIndex := lib.Bni(position[2]).Std().Int64()
			strategyAddress := ""
			strategyApy := lib.ZERO
			for _, s := range models.Strategies {
				if s.Index == strategyIndex {
					strategyAddress = s.Address
					strategyApy = s.Apy
				}
			}
			p := &models.Position{
				Index:       id.Int64(),
				Strategy:    strategyIndex,
				Token:       position[3].(common.Address).String(),
				Collateral:  lib.Bni(position[4]),
				Shares:      lib.Bni(position[6]),
				SharesValue: lib.Bni(client.Call(strategyAddress, "rate-uint256-uint256", lib.Bni(position[6]))[0]),
				Borrow:      lib.Bni(position[5]),
				BorrowValue: lib.Bni(position[5]).Mul(pool.Index).Div(lib.ONE6),
				Amount:      lib.Bni(position[7]),
				Created:     time.Unix(lib.Bni(position[1]).Std().Int64(), 0),
			}
			positions = append(positions, p)
			token := models.Tokens[p.Token]
			collateralValue := p.Collateral.Mul(tokenPrices[token.Address]).Div(lib.Bn(1, token.Decimals))
			positionLeverages[id.Int64()] = p.SharesValue.Mul(lib.ONE).Div(collateralValue)
			positionApys[id.Int64()] = strategyApy.Mul(positionLeverages[id.Int64()]).Div(lib.ONE)
			positionMaxBorrows[id.Int64()] = collateralValue.Mul(lib.Bn(10, 0)).Sub(p.BorrowValue)
			for _, c := range collaterals {
				if c.Token.Address == p.Token {
					c.PositionBalance = c.PositionBalance.Add(p.Collateral)
				}
			}
		}

		for _, c := range collaterals {
			c.UserBalance = client.CallUint(c.Token.Address, "balanceOf-address-uint256", address)
		}

		whitelisted = client.Call(models.AddressWhitelist, "check-address-bool", address)[0].(bool)
	}

	strategyPositions := map[int64][]*models.Position{}
	for _, p := range positions {
		strategyPositions[p.Strategy] = append(strategyPositions[p.Strategy], p)
	}

	// Edit position modal
	var position *models.Position
	if i := c.Param("p", ""); i != "" {
		index := lib.StringToInt(i)
		for _, p := range positions {
			if p.Index == index {
				position = p
				break
			}
		}
	}

	// New position modal
	tokenAddress := models.Collaterals[0].Address
	if c.Param("tab", "") == "new" {
		tokenAddress = c.Param("token", models.Collaterals[0].Address)
	} else if c.Param("tab", "") != "" {
		tokenAddress = position.Token
	}
	token := models.Tokens[tokenAddress]
	tokenPrice := tokenPrices[tokenAddress]
	balance := lib.ZERO
	tokenAllowance := lib.ZERO
	if address := c.GetCookie("address"); address != "" {
		balance = lib.Bni(client.Call(tokenAddress, "balanceOf-address-uint256", address)[0])
		tokenAllowance = lib.Bni(client.Call(tokenAddress, "allowance-address,address-uint256", address, models.AddressPositionManager)[0])
	}

	totalTvl := pool.Supply.Mul(lib.ONE12)
	totalValue := lib.ZERO
	totalBorrow := lib.ZERO
	totalCollateral := lib.ZERO
	totalApy := lib.ZERO
	for _, p := range positions {
		totalValue = totalValue.Add(p.SharesValue)
		totalBorrow = totalBorrow.Add(p.BorrowValue)
		totalCollateral = totalCollateral.Add(p.Collateral.Mul(tokenPrices[p.Token]).Div(lib.ONE))
		totalApy = totalApy.Add(positionApys[p.Index].Mul(p.SharesValue))
	}
	totalProfit := totalValue.Sub(totalBorrow)
	if !totalValue.Eq(lib.ZERO) {
		totalApy = totalApy.Div(totalValue)
	}
	for _, c := range collaterals {
		totalTvl = totalTvl.Add(c.Balance.Mul(c.Price).Div(lib.Bn(1, c.Token.Decimals)))
	}

	c.Render(200, "app/strategies", lib.J{
		"title":              "Farm",
		"strategies":         strategies,
		"positionManager":    models.AddressPositionManager,
		"tab":                tab,
		"strategyPositions":  strategyPositions,
		"positionLeverages":  positionLeverages,
		"positionApys":       positionApys,
		"positionMaxBorrows": positionMaxBorrows,
		"whitelisted":        whitelisted,

		"earnApy":        pool.Rate,
		"strategy":       strategy,
		"balance":        balance,
		"token":          token,
		"tokenPrice":     tokenPrice,
		"tokenAllowance": tokenAllowance,
		"borrow":         c.Param("borrow", ""),
		"collateral":     c.Param("collateral", ""),
		"collaterals":    collaterals,

		"position": position,

		"totalTvl":        totalTvl,
		"totalValue":      totalValue,
		"totalBorrow":     totalBorrow,
		"totalProfit":     totalProfit,
		"totalCollateral": totalCollateral,
		"totalApy":        totalApy,
	})
}

func AppStrategy(c *lib.Ctx) {
	strategies := models.Strategies
	c.Cache.Try("strategies", &strategies, 5*time.Minute, cacheStrategies(c))
	var strategy *models.StrategyInfo
	slug := c.Param("slug", "")
	for _, s := range strategies {
		if s.Slug == slug {
			strategy = s
			break
		}
	}
	if strategy == nil {
		c.Render(404, "other/404", lib.J{})
		return
	}
	c.Render(200, "app/strategy", lib.J{
		"title":    strategy.Protocol + ": " + strategy.Name,
		"strategy": strategy,
	})
}

func AppLend(c *lib.Ctx) {
	client := c.Server.ChainClients[models.DefaultChainId]
	pool := &models.PoolInfo{}
	c.Cache.Try("pool", pool, 60*time.Second, cachePool(c))

	address := c.GetCookie("address")
	balanceShares := lib.ZERO
	balanceLent := lib.ZERO
	balanceAsset := lib.ZERO
	allowance := lib.ZERO
	if address != "" {
		balanceShares = client.CallUint(pool.Address, "balanceOf-address-uint256", address)
		balanceLent = balanceShares.Mul(pool.Index).Div(lib.ONE)
		balanceAsset = client.CallUint(pool.Asset, "balanceOf-address-uint256", address)
		allowance = lib.Bni(client.Call(pool.Asset, "allowance-address,address-uint256", address, pool.Address)[0])
	}

	c.Render(200, "app/lend", lib.J{
		"title":         "Earn",
		"pool":          models.Pools[0],
		"balanceShares": balanceShares,
		"balanceLent":   balanceLent,
		"balanceAsset":  balanceAsset,
		"allowance":     allowance,
		"tab":           c.Param("tab", "deposit"),
	})
}

func AppStaking(c *lib.Ctx) {
	client := c.Server.ChainClients[models.DefaultChainId]
	total := lib.Bni(client.Call(models.AddressXrdo, "plugins-uint256-uint256", lib.Bn(0, 0))[0])

	address := c.GetCookie("address")
	balance := lib.ZERO
	deposited := lib.ZERO
	allowance := lib.ZERO
	dividendsRdo := lib.ZERO
	dividendsUsdc := lib.ZERO
	if address != "" {
		balance = lib.Bni(client.Call(models.AddressRdo, "balanceOf-address-uint256", address)[0])
		deposited = lib.Bni(client.Call(models.AddressXrdo, "getUser-address,uint256-uint256,uint256,uint256[]", address, lib.Bn(3, 0))[2].([]*big.Int)[0])
		allowance = lib.Bni(client.Call(models.AddressRdo, "allowance-address,address-uint256", address, models.AddressXrdo)[0])
		dividendsRdo = lib.Bni(client.Call(models.AddressTokenStakingDividends, "claimable-address,uint256-uint256", address, lib.Bn(0, 0))[0])
		dividendsUsdc = lib.Bni(client.Call(models.AddressTokenStakingDividends, "claimable-address,uint256-uint256", address, lib.Bn(1, 0))[0])
	}
	c.Render(200, "app/staking", lib.J{
		"title":                 "Silo",
		"rdo":                   models.AddressRdo,
		"xrdo":                  models.AddressXrdo,
		"tokenStakingDividends": models.AddressTokenStakingDividends,
		"apy":                   lib.Bn(239824, 36).Div(total),
		"total":                 total,
		"balance":               balance,
		"deposited":             deposited,
		"allowance":             allowance,
		"dividendsRdo":          dividendsRdo,
		"dividendsUsdc":         dividendsUsdc,
		"tab":                   c.Param("tab", "deposit"),
	})
}

func AppRewards(c *lib.Ctx) {
	client := c.Server.ChainClients[models.DefaultChainId]
	data := struct {
		Farm *lib.BigInt
		Earn *lib.BigInt
	}{}
	c.Cache.Try("rewards", &data, 15*time.Minute, func() interface{} {
		arbPrice := lib.Bni(client.Call("0xb2A824043730FE05F3DA2efaFa1CBbe83fa548D6", "latestAnswer--int256")[0]).Mul(lib.ONE10)
		farmSupply := lib.Bni(client.Call("0x72b9E266b4F531A5a41fE56D5B2ae1Bafba196c0", "tvl--uint256")[0])
		earnSupply := lib.Bni(client.Call(models.AddressPool, "getTotalLiquidity--uint256")[0]).Mul(lib.ONE12)
		data.Farm = lib.Bn(10000, 18).Mul(arbPrice).Div(farmSupply)
		data.Earn = lib.Bn(90000, 18).Mul(arbPrice).Div(earnSupply)
		return data
	})

	address := c.GetCookie("address")
	weeks := []lib.J{
		{"id": "0", "name": "Jan 15 - Jan 21, 2024"},
		{"id": "1", "name": "Jan 22 - Jan 28, 2024"},
		{"id": "2", "name": "Jan 29 - Feb 04, 2024"},
		{"id": "3", "name": "Feb 05 - Feb 11, 2024"},
		{"id": "4", "name": "Feb 12 - Feb 18, 2024"},
		{"id": "5", "name": "Feb 19 - Feb 25, 2024"},
		{"id": "6", "name": "Feb 26 - Mar 3, 2024"},
		{"id": "7", "name": "Bulls: Season 1"},
		{"id": "8", "name": "Bulls: Season 1: Adjustment"},
	}
	indexes := []*big.Int{}
	claimed := []*big.Int{}
	for i := range weeks {
		indexes = append(indexes, big.NewInt(int64(i)))
		claimed = append(claimed, big.NewInt(0))
	}
	if address != "" {
		claimed = client.Call(models.AddressSTIPDistributor, "getClaimed-uint256[],address-uint256[]", indexes, address)[0].([]*big.Int)
	}
	for i, w := range weeks {
		data := lib.J{}
		bs, err := c.Server.FS.ReadFile(fmt.Sprintf("assets/stip/%s.json", w.Get("id")))
		lib.Check(err)
		lib.Check(json.Unmarshal(bs, &data))
		w.Set("data", data)
		w.Set("claimed", lib.Bnw(claimed[i]))
		w.Set("claimable", lib.ZERO)
		for _, u := range data["users"].([]interface{}) {
			uu := u.(map[string]interface{})
			if uu["user"].(string) == address {
				w.Set("claimable", lib.Bns(uu["amount"].(string)))
				w.Set("proof", uu["proof"])
				break
			}
		}
	}
	c.Render(200, "app/rewards", lib.J{
		"title":           "Rewards",
		"stipDistributor": models.AddressSTIPDistributor,
		"data":            data,
		"weeks":           weeks,
	})
}

func AppVesting(c *lib.Ctx) {
	client := c.Server.ChainClients[models.DefaultChainId]
	sourceNames := map[string]string{
		"1": "xRDO Redeem",
		"2": "Public Sale",
		"3": "Private Round",
		"4": "Private xRDO exits",
		"5": "KOL",
	}
	tab := c.Param("tab", "active")
	vestings := []lib.J{}
	address := c.GetCookie("address")
	if address != "" {
		count := lib.Bni(client.Call(models.AddressVester, "schedulesCount-address-uint256", address)[0])
		schedules := client.Call(models.AddressVester, "getSchedules-address,uint256,uint256-uint256[],uint256[],uint256[]", address, big.NewInt(0), count)
		amounts := schedules[0].([]*big.Int)
		claimeds := schedules[1].([]*big.Int)
		availables := schedules[2].([]*big.Int)
		infos := client.Call(models.AddressVester, "getSchedulesInfo-address,uint256,uint256-uint256[],address[],uint256[],uint256[],uint256[]", address, big.NewInt(0), count)
		sources := infos[0].([]*big.Int)
		times := infos[3].([]*big.Int)
		starts := infos[4].([]*big.Int)
		for i := 0; i < len(amounts); i++ {
			if lib.Bnw(amounts[i]).Eq(lib.ZERO) {
				break
			}
			done := lib.Bnw(amounts[i]).Eq(lib.Bnw(claimeds[i]))
			if (tab == "active" && done) || (tab == "completed" && !done) {
				continue
			}

			vesting := lib.J{
				"index":     i,
				"source":    sourceNames[sources[i].String()],
				"claimed":   lib.Bnw(claimeds[i]),
				"amount":    lib.Bnw(amounts[i]),
				"available": lib.Bnw(availables[i]),
				"start":     time.Unix(lib.Bnw(starts[i]).Std().Int64(), 0),
				"end":       time.Unix(lib.Bnw(starts[i]).Add(lib.Bnw(times[i])).Std().Int64(), 0),
			}
			vestings = append([]lib.J{vesting}, vestings...)
		}
	}
	c.Render(200, "app/vesting", lib.J{
		"title":    "Vesting",
		"tab":      tab,
		"vestor":   models.AddressVester,
		"vestings": vestings,
		//"balanceAsset": balanceAsset,
	})
}

func AppAnalytics(c *lib.Ctx) {
	client := c.Server.ChainClients[models.DefaultChainId]
	data := struct {
		TokenPrice             *lib.BigInt
		MarketCap              *lib.BigInt
		MarketCapFullyDilluted *lib.BigInt
		SupplyCirculating      *lib.BigInt
		SupplyMax              *lib.BigInt
		SupplyTotal            *lib.BigInt
		SupplyPOL              *lib.BigInt
		SupplyXrdo             *lib.BigInt
		SupplyEcosystem        *lib.BigInt
		SupplyPartners         *lib.BigInt
		SupplyTeam             *lib.BigInt
		SupplyMultisig         *lib.BigInt
		SupplyDeployer         *lib.BigInt
		Largest                []*models.Position
		Profit                 []*models.Position
		Danger                 []*models.Position
	}{}
	c.Cache.Try("analytics", &data, 15*time.Minute, func() interface{} {
		data.TokenPrice = lib.Bni(client.Call("0x309349d5D02C6f8b50b5040e9128E1A8375042D7", "latestAnswer--int256")[0])

		data.SupplyMax = lib.Bn(100_000_000, 18)
		rdoInLp2 := lib.Bni(client.Call(models.AddressRdo, "balanceOf-address-uint256", models.AddressLPToken2)[0])
		msOwnedLp2 := lib.Bni(client.Call(models.AddressLPToken2, "balanceOf-address-uint256", models.AddressMultisigCamelot)[0])
		lp2TotalSupply := lib.Bni(client.Call(models.AddressLPToken2, "totalSupply--uint256")[0])
		data.SupplyPOL = rdoInLp2.Mul(msOwnedLp2).Div(lp2TotalSupply)
		data.SupplyXrdo = lib.Bni(client.Call(models.AddressRdo, "balanceOf-address-uint256", models.AddressXrdo)[0])
		data.SupplyTeam = lib.Bni(client.Call(models.AddressRdo, "balanceOf-address-uint256", models.AddressMultisigTeam)[0])
		data.SupplyEcosystem = lib.Bni(client.Call(models.AddressRdo, "balanceOf-address-uint256", "0x91E375808aD4DCE30461c852B3C64a6a13981d3C")[0])
		data.SupplyPartners = lib.Bni(client.Call(models.AddressRdo, "balanceOf-address-uint256", "0x6Bdee28E211BeD4cC0BEB6276A4dbbc108cb1878")[0])
		data.SupplyMultisig = lib.Bni(client.Call(models.AddressRdo, "balanceOf-address-uint256", models.AddressMultisig)[0])
		data.SupplyDeployer = lib.Bni(client.Call(models.AddressRdo, "balanceOf-address-uint256", "0x20dE070F1887f82fcE2bdCf5D6d9874091e6FAe9")[0])
		data.SupplyTotal = lib.Bni(client.Call(models.AddressRdo, "totalSupply--uint256")[0])
		data.SupplyCirculating = data.SupplyTotal.
			Sub(data.SupplyPOL).
			Sub(data.SupplyXrdo).
			Sub(data.SupplyTeam).
			Sub(data.SupplyEcosystem).
			Sub(data.SupplyPartners).
			Sub(data.SupplyMultisig).
			Sub(data.SupplyDeployer)
		data.MarketCap = data.TokenPrice.Mul(data.SupplyCirculating).Div(lib.ONE)
		data.MarketCapFullyDilluted = data.TokenPrice.Mul(data.SupplyTotal).Div(lib.ONE)

		c.DB.AllWhere(&data.Largest, `1=1 order by shares_value desc limit 10`)
		c.DB.AllWhere(&data.Profit, `amount > 0 and shares_value > 0 order by ((shares_value - ((borrow_value + amount) * price / 1e6)) * 1e18 / (amount * price / 1e6)) desc limit 10`)
		c.DB.AllWhere(&data.Danger, `shares_value > 10e18 and life > 1e18 order by life asc limit 10`)
		return data
	})
	c.Render(200, "app/analytics", lib.J{
		"title": "Analytics",
		"data":  data,
	})
}
