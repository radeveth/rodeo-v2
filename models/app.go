package models

import "app/lib"

type TokenInfo struct {
	Address  string
	Symbol   string
	Icon     string
	Decimals int64
	Oracle   string
}

type PoolInfo struct {
	Asset   string `json:"asset"`
	Address string `json:"address"`
	Slug    string `json:"slug"`

	RateModelKink *lib.BigInt `json:"rateModelKink"`
	RateModelBase *lib.BigInt `json:"rateModelBase"`
	RateModelLow  *lib.BigInt `json:"rateModelLow"`
	RateModelHigh *lib.BigInt `json:"rateModelHigh"`

	Paused    bool        `json:"paused"`
	BorrowMin *lib.BigInt `json:"borrowMin"`
	Cap       *lib.BigInt `json:"cap"`
	Index     *lib.BigInt `json:"index"`
	Shares    *lib.BigInt `json:"shares"`
	Supply    *lib.BigInt `json:"supply"`
	Borrow    *lib.BigInt `json:"borrow"`
	Rate      *lib.BigInt `json:"rate"`
	Price     *lib.BigInt `json:"price"`

	SupplyRate  *lib.BigInt `json:"supplyRate"`
	Utilisation *lib.BigInt `json:"utilisation"`
	ArbRate     *lib.BigInt `json:"arbRate"`
}

type StrategyInfo struct {
	Index          int64           `json:"index"`
	Slug           string          `json:"slug"`
	Name           string          `json:"name"`
	Protocol       string          `json:"protocol"`
	Icon           string          `json:"icon"`
	ApyType        string          `json:"apyType"`
	ApyId          string          `json:"apyId"`
	Description    string          `json:"description"`
	Fees           string          `json:"fees"`
	Underlying     []StrategyToken `json:"underlying"`
	IsHidden       bool            `json:"isHidden"`
	IsEarnDisabled bool            `json:"isEarnDisabled"`

	Address         string      `json:"address"`
	Status          int64       `json:"status"`
	Cap             *lib.BigInt `json:"cap"`
	Tvl             *lib.BigInt `json:"tvl"`
	TvlTotal        *lib.BigInt `json:"tvlTotal"`
	Apy             *lib.BigInt `json:"apy"`
	ApyWithLeverage *lib.BigInt `json:"apyWithLeverage"`
}

type StrategyToken struct {
	Ratio   float64 `json:"ratio"`
	Address string  `json:"address"`
}

type VaultInfo struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Address     string `json:"address"`
	Strategy    string `json:"strategy"`
	Asset       string `json:"asset"`
	AssetOracle string `json:"assetOracle"`
	AssetSymbol string `json:"assetSymbol"`

	TargetLeverage *lib.BigInt `json:"targetLeverage"`
	DepositFee     float64     `json:"depositFee"`
	WithdrawFee    float64     `json:"withdrawFee"`
	ManagementFee  float64     `json:"managementFee"`

	TotalAssets *lib.BigInt `json:"totalAssets"`
	TotalSupply *lib.BigInt `json:"totalSupply"`
	APY         *lib.BigInt `json:"apy"`
	BaseAPY     *lib.BigInt `json:"baseApy"`
	Leverage    *lib.BigInt `json:"leverage"`
	Cap         *lib.BigInt `json:"cap"`
	AssetPrice  *lib.BigInt `json:"assetPrice"`
}

var Tokens = map[string]*TokenInfo{
	"0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8": &TokenInfo{
		Address:  "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8",
		Symbol:   "USDC.e",
		Icon:     "/assets/assets/usdc.svg",
		Decimals: 6,
		Oracle:   "0x50834F3163758fcC1Df9973b6e91f0F0F0434aD3",
	},
	"0xaf88d065e77c8cC2239327C5EDb3A432268e5831": &TokenInfo{
		Address:  "0xaf88d065e77c8cC2239327C5EDb3A432268e5831",
		Symbol:   "USDC",
		Icon:     "/assets/assets/usdc.svg",
		Decimals: 6,
		Oracle:   "0x50834F3163758fcC1Df9973b6e91f0F0F0434aD3",
	},
	"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": &TokenInfo{
		Address:  "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
		Symbol:   "WETH",
		Icon:     "/assets/assets/eth.svg",
		Decimals: 18,
		Oracle:   "0x639Fe6ab55C921f74e7fac1ee960C0B6293ba612",
	},
	"0x5979D7b546E38E414F7E9822514be443A4800529": &TokenInfo{
		Address:  "0x5979D7b546E38E414F7E9822514be443A4800529",
		Symbol:   "wstETH",
		Icon:     "/assets/assets/wsteth.png",
		Decimals: 18,
		Oracle:   "0xC75B29Cfd5244FBf55c5567FbF20a3C2D83c8A80",
	},
	"0xEC70Dcb4A1EFa46b8F2D97C310C9c4790ba5ffA8": &TokenInfo{
		Address:  "0xEC70Dcb4A1EFa46b8F2D97C310C9c4790ba5ffA8",
		Symbol:   "rETH",
		Icon:     "/assets/assets/reth.png",
		Decimals: 18,
		Oracle:   "0x09E152B25316e574579bEcC3C760D3730A8B6AC3",
	},
}

type Collateral struct {
	Token           *TokenInfo
	Price           *lib.BigInt
	Balance         *lib.BigInt
	UserBalance     *lib.BigInt
	PositionBalance *lib.BigInt
	Cap             *lib.BigInt
}

var Collaterals = []*TokenInfo{
	Tokens["0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"],
	//Tokens["0xaf88d065e77c8cC2239327C5EDb3A432268e5831"],
	Tokens["0x82aF49447D8a07e3bd95BD0d56f35241523fBab1"],
	Tokens["0x5979D7b546E38E414F7E9822514be443A4800529"],
	Tokens["0xEC70Dcb4A1EFa46b8F2D97C310C9c4790ba5ffA8"],
}

var Pools = []*PoolInfo{
	&PoolInfo{
		Asset:   "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8",
		Address: AddressPool,
		Slug:    "usdc",
	},
}

var Strategies = []*StrategyInfo{
	&StrategyInfo{
		Index:       1,
		Slug:        "camelotv3-weth-usdc",
		Name:        "ETH/USDC",
		Protocol:    "Camelot V3",
		Icon:        "/assets/protocols/camelot.svg",
		ApyType:     "defillama",
		ApyId:       "a51d982c-72ad-43b1-9cb2-0273c46655f3",
		Description: "Mint Camelot V3 LP compounding rewards from the NFT pool (GRAIL) and, when available, extra Nitro pool rewards.",
		Fees:        "0.5%",
		Underlying:  []StrategyToken{{Ratio: 50, Address: ""}, {Ratio: 50, Address: ""}},
		Address:     "0xE4cFFC97C7EF18cb9e33e787CF70c2dA8eBf1a59",
	},
	/*
		&StrategyInfo{
			Index:       1,
			Slug:        "gmxv2-btc",
			Name:        "BTC",
			Protocol:    "GMX V2",
			Icon:        "/assets/protocols/gmx.png",
			ApyType:     "defillama",
			ApyId:       "5b8c0691-b9ff-4d82-97e4-19a1247e6dbf",
			Description: "",
			Fees:        "0.5%",
			Underlying:  []StrategyToken{{Ratio: 50, Address: "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f"}, {Ratio: 50, Address: "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"}},

			Address: "0x11bB5c95E45f2428B2BCeC2797C5c869894ccec7",
		},
	*/
}

var Vaults = []*VaultInfo{
	&VaultInfo{
		Name:           "AAVE Looped wstETH",
		Slug:           "loop-aave-wsteth",
		Address:        "0xe00A5490149D154c31074598B58E36451D0260BC",
		Strategy:       "0xEe7933989950A5B67DfF80643C9B07E2ff50446A",
		Asset:          "0x5979D7b546E38E414F7E9822514be443A4800529",
		AssetOracle:    "0xC75B29Cfd5244FBf55c5567FbF20a3C2D83c8A80",
		AssetSymbol:    "wstETH",
		TargetLeverage: lib.Bn(5, 18),
		DepositFee:     0.5,
		WithdrawFee:    0.5,
		ManagementFee:  1,
	},
}
