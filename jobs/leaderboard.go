package jobs

import (
	"app/lib"
	"app/models"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var _ = lib.RegisterSchedule("leaderboard", time.Hour)

var _ = lib.RegisterJob("leaderboard-backfill", func(c *lib.Ctx, args lib.J) {

	client := c.Server.ChainClients[models.DefaultChainId]

	points := map[string]*lib.BigInt{}

	credit := func(balances map[string]*lib.BigInt) {
		for k, v := range balances {
			if k == lib.ADDRESS_ZERO || v.Lte(lib.Bn(1, 18)) {
				delete(balances, k)
				continue
			}
		}
		for k, v := range balances {
			if _, ok := points[k]; !ok {
				points[k] = lib.ZERO
			}
			points[k] = points[k].Add(v)
		}
	}

	start := 192404756
	end := 195600000
	lm0 := "0x3A039A4125E8B8012CF3394eF7b8b02b739900b1"
	lm1 := "0x3aEe6cA602C060883201B89c64cb5F782F964879"
	logs := client.FilterLogsBlock(models.AddressPool, []string{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"}, int64(end))
	for block := start; block < end; block += 14150 {
		fmt.Printf("%v / %.2f%%\n", block, float64(block-start)*100/float64(end-start))

		// Lending
		balances := map[string]*lib.BigInt{}
		for _, l := range logs {
			if int(l.BlockNumber) > block {
				continue
			}
			inp := common.HexToAddress("0x" + l.Topics[1].Hex()[26:]).Hex()
			out := common.HexToAddress("0x" + l.Topics[2].Hex()[26:]).Hex()
			amount := lib.Bnw(new(big.Int).SetBytes(l.Data)).Mul(lib.Bn(107, 10))
			if inp == lm0 || out == lm0 || inp == lm1 || out == lm1 {
				continue
			}
			if balances[inp] == nil {
				balances[inp] = lib.ZERO
			}
			if balances[out] == nil {
				balances[out] = lib.ZERO
			}
			balances[inp] = balances[inp].Sub(amount)
			balances[out] = balances[out].Add(amount)
		}
		credit(balances)

		// Farming
		balances = map[string]*lib.BigInt{}
		data := client.CallWithBlock(big.NewInt(int64(block)), "0x907F2323d58A7f0500EDefa0e146b4D1f38D865C", "get-uint256-address[],uint256[]", big.NewInt(3000))
		positionBalances := data[1].([]*big.Int)
		for i, a := range data[0].([]common.Address) {
			if balances[a.Hex()] == nil {
				balances[a.Hex()] = lib.ZERO
			}
			balances[a.Hex()] = balances[a.Hex()].Add(lib.Bnw(positionBalances[i]))
		}
		credit(balances)
	}

	total := lib.ZERO
	for _, v := range points {
		total = total.Add(v)
	}
	for k, v := range points {
		fmt.Printf("%s,%s\n", k, v.Mul(lib.Bn(240000, 0)).Div(total).String())
	}
	fmt.Printf("total,%s\n", total.String())
})

var _ = lib.RegisterJob("leaderboard", func(c *lib.Ctx, args lib.J) {
	if time.Now().Unix() > 1711695600 {
		return
	}

	client := c.Server.ChainClients[models.DefaultChainId]

	// Lending
	lm0 := "0x3A039A4125E8B8012CF3394eF7b8b02b739900b1"
	lm1 := "0x3aEe6cA602C060883201B89c64cb5F782F964879"
	logs := client.FilterLogs(models.AddressPool, []string{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"})
	balances := map[string]*lib.BigInt{}
	for _, l := range logs {
		inp := common.HexToAddress("0x" + l.Topics[1].Hex()[26:]).Hex()
		out := common.HexToAddress("0x" + l.Topics[2].Hex()[26:]).Hex()
		amount := lib.Bnw(new(big.Int).SetBytes(l.Data)).Mul(lib.Bn(107, 10))
		if inp == lm0 || out == lm0 || inp == lm1 || out == lm1 {
			continue
		}
		if balances[inp] == nil {
			balances[inp] = lib.ZERO
		}
		if balances[out] == nil {
			balances[out] = lib.ZERO
		}
		balances[inp] = balances[inp].Sub(amount)
		balances[out] = balances[out].Add(amount)
	}
	cleanAndCredit(c, client, balances, "Lending")

	// Farming
	balances = map[string]*lib.BigInt{}
	data := client.Call("0x907F2323d58A7f0500EDefa0e146b4D1f38D865C", "get-uint256-address[],uint256[]", big.NewInt(3000))
	positionBalances := data[1].([]*big.Int)
	for i, a := range data[0].([]common.Address) {
		if balances[a.Hex()] == nil {
			balances[a.Hex()] = lib.ZERO
		}
		balances[a.Hex()] = balances[a.Hex()].Add(lib.Bnw(positionBalances[i]))
	}
	cleanAndCredit(c, client, balances, "Farming")
})

func cleanAndCredit(c *lib.Ctx, client *lib.ChainClient, balances map[string]*lib.BigInt, action string) {
	// Clean
	for k, v := range balances {
		if k == lib.ADDRESS_ZERO || v.Lte(lib.Bn(1, 18)) {
			delete(balances, k)
			continue
		}
	}

	// xRDO Balances
	rdoPrice := client.CallUint("0x309349d5D02C6f8b50b5040e9128E1A8375042D7", "latestAnswer--int256")
	xrdoBalances := map[string]*lib.BigInt{}
	for k := range balances {
		xrdoBalances[k] = client.CallUint(models.AddressXrdo, "balanceOf-address-uint256", k).Mul(rdoPrice).Div(lib.ONE)
	}

	for k, v := range balances {
		boost := xrdoBalances[k].Mul(lib.Bn(10, 0)).Mul(lib.ONE).Div(v)
		if boost.Gt(lib.Bn(10, 18)) {
			boost = lib.Bn(10, 18)
		}
		points := v.Mul(lib.Bn(100, 0)).Div(lib.Bn(24, 0)).Mul(boost.Add(lib.ONE)).Div(lib.Bn(1000, 18+18)).Std().Int64()
		if points <= 0 {
			continue
		}
		u := &models.LeaderboardUser{}
		c.DB.FirstWhere(u, "address = $1", k)
		if u.ID == "" {
			u.ID = lib.NewID()
			u.Code = lib.NewRandomID()[0:10]
			u.Address = k
			u.Created = time.Now()
			c.DB.Put(u)
			models.LeaderboardPointCredit(c, u.ID, "Connect", u.Address, 100)
		}
		models.LeaderboardPointCredit(c, u.ID, action, v.Mul(lib.ONE12).String(), points)
		lib.LogInfo("leaderboard", lib.J{"action": action, "address": k, "value": v.Float() / 1e18, "points": points, "boost": boost.Float() / 1e18, "xrdo": xrdoBalances[k].Float() / 1e18})
	}
}
