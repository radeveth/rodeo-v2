package lib

import (
	"context"
	"crypto/ecdsa"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var ZERO = Bn(0, 0)
var YEAR = Bn(365*24*60*60, 0)
var ONE = Bn(1, 18)
var ONE6 = Bn(1, 6)
var ONE8 = Bn(1, 8)
var ONE10 = Bn(1, 10)
var ONE12 = Bn(1, 12)
var ADDRESS_ZERO = "0x0000000000000000000000000000000000000000"

type BigInt big.Int

func (b *BigInt) Value() (driver.Value, error) {
	if b != nil {
		return (*big.Int)(b).String(), nil
	}
	return nil, nil
}

func (b *BigInt) Scan(value interface{}) error {
	if value == nil {
		b = nil
	}
	switch t := value.(type) {
	case []uint8:
		f, _, err := big.ParseFloat(string(value.([]uint8)), 10, 0, big.ToNearestEven)
		if err != nil {
			return fmt.Errorf("failed to load value to []uint8: %v", value)
		}
		f.Int((*big.Int)(b))
	default:
		return fmt.Errorf("Could not scan type %T into BigInt", t)
	}
	return nil
}

func (b *BigInt) MarshalJSON() ([]byte, error) {
	if b == nil {
		return []byte(`"0"`), nil
	}
	return json.Marshal(b.String())
}

func (b *BigInt) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if _, ok := (*big.Int)(b).SetString(s, 10); !ok {
		return fmt.Errorf("failed to parse bigint %s", s)
	}
	return nil
}

func (b *BigInt) Eq(a *BigInt) bool {
	return (*big.Int)(b).Cmp((*big.Int)(a)) == 0
}

func (b *BigInt) Gt(a *BigInt) bool {
	return (*big.Int)(b).Cmp((*big.Int)(a)) == 1
}

func (b *BigInt) Lt(a *BigInt) bool {
	return (*big.Int)(b).Cmp((*big.Int)(a)) == -1
}

func (b *BigInt) Lte(a *BigInt) bool {
	cmp := (*big.Int)(b).Cmp((*big.Int)(a))
	return cmp == 0 || cmp == -1
}

func (b *BigInt) Add(a *BigInt) *BigInt {
	return (*BigInt)((&big.Int{}).Add((*big.Int)(b), (*big.Int)(a)))
}

func (b *BigInt) Sub(a *BigInt) *BigInt {
	return (*BigInt)((&big.Int{}).Sub((*big.Int)(b), (*big.Int)(a)))
}

func (b *BigInt) Mul(a *BigInt) *BigInt {
	return (*BigInt)((&big.Int{}).Mul((*big.Int)(b), (*big.Int)(a)))
}

func (b *BigInt) Div(a *BigInt) *BigInt {
	return (*BigInt)((&big.Int{}).Div((*big.Int)(b), (*big.Int)(a)))
}

func (b *BigInt) Std() *big.Int {
	return (*big.Int)(b)
}

func (b *BigInt) Float() float64 {
	return StringToFloat(b.String())
}

func (b *BigInt) String() string {
	return (*big.Int)(b).String()
}

func Bn(num int64, base int64) *BigInt {
	x := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(base), nil)
	n := big.NewInt(num)
	return (*BigInt)(n.Mul(n, x))
}

func Bnf(num float64, base int64) *BigInt {
	x := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(base), nil)
	n := big.NewInt(int64(num * 1000000000))
	n.Mul(n, x)
	n.Div(n, big.NewInt(1000000000))
	return (*BigInt)(n)
}

func Bns(s string) *BigInt {
	b := new(big.Int)
	_, ok := b.SetString(s, 10)
	if !ok {
		panic(fmt.Errorf("Bns: error parsing bigint: %s", s))
	}
	return (*BigInt)(b)
}

func Bnw(b *big.Int) *BigInt {
	return (*BigInt)(b)
}

func Bni(b interface{}) *BigInt {
	return (*BigInt)(b.(*big.Int))
}

type ChainClient struct {
	Client        *ethclient.Client
	ChainID       *big.Int
	WalletAddress common.Address
	WalletKey     *ecdsa.PrivateKey
}

func NewChainClient(chainId int64) *ChainClient {
	var err error
	c := &ChainClient{}
	c.ChainID = big.NewInt(chainId)
	c.Client, err = ethclient.Dial(Env("RPC_URL_"+strconv.FormatInt(chainId, 10), "https://arb1.arbitrum.io/rpc"))
	Check(err)
	c.WalletKey = crypto.ToECDSAUnsafe(common.FromHex(Env("PRIVATE_KEY", "")))
	c.WalletAddress = crypto.PubkeyToAddress(c.WalletKey.PublicKey)
	return c
}

func (c *ChainClient) CallUint(to string, fn string, args ...interface{}) *BigInt {
	return Bni(c.Call(to, fn, args...)[0])
}

func (c *ChainClient) Call(to string, fn string, args ...interface{}) []interface{} {
	return c.CallWithBlock(nil, to, fn, args...)
}

func (c *ChainClient) CallWithBlock(block *big.Int, to string, fn string, args ...interface{}) []interface{} {
	// parse fn string
	var err error
	parts := strings.Split(fn, "-")
	name := parts[0]
	isWrite := name[0] == '+'
	if isWrite {
		name = name[1:]
	}
	inputs := strings.Split(parts[1], ",")
	outputs := strings.Split(parts[2], ",")

	// build abi
	mInputs := abi.Arguments{}
	mOutputs := abi.Arguments{}
	for j, i := range inputs {
		if i == "" {
			continue
		}
		t, err := abi.NewType(i, i, nil)
		Check(err)
		mInputs = append(mInputs, abi.Argument{Type: t})
		if i == "address" {
			if a, ok := args[j].(string); ok {
				args[j] = common.HexToAddress(a)
			}
		}
		if i == "bytes" {
			if b, ok := args[j].(string); ok {
				args[j] = hexutil.MustDecode(b)
			}
		}
		if strings.HasPrefix(i, "uint") || strings.HasPrefix(i, "int") {
			if n, ok := args[j].(*BigInt); ok {
				args[j] = n.Std()
			}
		}
	}
	for _, o := range outputs {
		if o == "" {
			continue
		}
		t, err := abi.NewType(o, o, nil)
		Check(err)
		mOutputs = append(mOutputs, abi.Argument{Type: t})
	}
	method := abi.NewMethod(name, name, abi.Function, "view", false, false, mInputs, mOutputs)
	fnabi := &abi.ABI{}
	fnabi.Methods = map[string]abi.Method{name: method}

	// build message
	message := ethereum.CallMsg{}
	bto := common.HexToAddress(to)
	message.To = &bto
	message.Data, err = fnabi.Pack(name, args...)
	Check(err)

	if !isWrite {
		// call and unpack result
		bs, err := c.Client.CallContract(context.Background(), message, block)
		Check(err)
		result, err := fnabi.Unpack(name, bs)
		Check(err)
		return result
	} else {
		message.From = c.WalletAddress
		data := &types.DynamicFeeTx{}
		data.Gas, err = c.Client.EstimateGas(context.Background(), message)
		if err != nil {
			var trace interface{}
			c.Client.Client().Call(&trace, "debug_traceCall", J{
				"from": c.WalletAddress.String(),
				"to":   to,
				"data": hexutil.Encode(message.Data),
			}, "latest", J{"tracer": "callTracer"})
			bs, _ := json.MarshalIndent(trace, "", "  ")
			fmt.Println(string(bs))
			panic(fmt.Errorf("error estimating gas: %w: %s %s %v", err, to, fn, args))
		}
		data.Nonce, err = c.Client.PendingNonceAt(context.Background(), message.From)
		Check(err)
		data.GasTipCap = Bn(1, 0).Std()
		data.GasFeeCap = Bn(200000000, 0).Std()
		data.ChainID = c.ChainID
		data.To = message.To
		data.Data = message.Data
		tx := types.MustSignNewTx(c.WalletKey, types.LatestSignerForChainID(c.ChainID), data)
		if IsProduction() {
			Check(c.Client.SendTransaction(context.Background(), tx))
		}
		return []interface{}{tx.Hash().Hex()}
	}
}

func (c *ChainClient) FilterLogs(address string, topics []string) []types.Log {
	etopics := [][]common.Hash{}
	for _, t := range topics {
		etopics = append(etopics, []common.Hash{common.HexToHash(t)})
	}
	logs, err := c.Client.FilterLogs(context.Background(), ethereum.FilterQuery{
		FromBlock: big.NewInt(39117212),
		Addresses: []common.Address{common.HexToAddress(address)},
		Topics:    etopics,
	})
	Check(err)
	return logs
}

func (c *ChainClient) FilterLogsBlock(address string, topics []string, toBlock int64) []types.Log {
	etopics := [][]common.Hash{}
	for _, t := range topics {
		etopics = append(etopics, []common.Hash{common.HexToHash(t)})
	}
	logs, err := c.Client.FilterLogs(context.Background(), ethereum.FilterQuery{
		FromBlock: big.NewInt(39117212),
		ToBlock:   big.NewInt(toBlock),
		Addresses: []common.Address{common.HexToAddress(address)},
		Topics:    etopics,
	})
	Check(err)
	return logs
}
