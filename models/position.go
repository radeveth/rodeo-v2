package models

import (
	"app/lib"
	"time"
)

type Earn struct {
	Id   int64
	Last time.Time
	Next time.Time
}

type Price struct {
	Id    string      `json:"asset"`
	Price *lib.BigInt `json:"price"`
	Time  time.Time   `json:"time"`
}

type Position struct {
	Id          int64       `json:"id"`
	Chain       int64       `json:"chain"`
	Index       int64       `json:"index"`
	Pool        string      `json:"pool"`
	Strategy    int64       `json:"strategy"`
	Shares      *lib.BigInt `json:"shares"`
	Borrow      *lib.BigInt `json:"borrow"`
	SharesValue *lib.BigInt `json:"shares_value"`
	BorrowValue *lib.BigInt `json:"borrow_value"`
	Life        *lib.BigInt `json:"life"`
	Amount      *lib.BigInt `json:"amount"`
	Price       *lib.BigInt `json:"price"`
	Created     time.Time   `json:"created"`
	Updated     time.Time   `json:"updated"`

	Token      string      `json:"token"`
	Collateral *lib.BigInt `json:"collateral"`
}

func (p *Position) CollateralToken() *TokenInfo {
	return Tokens[p.Token]
}

func (p *Position) Profit() *lib.BigInt {
	return p.SharesValue.Sub(p.BorrowValue.Mul(lib.ONE12)).Sub(p.Amount.Mul(lib.ONE12))
}

func (p *Position) ProfitPercent() *lib.BigInt {
	if p.Amount.Eq(lib.ZERO) {
		return lib.ZERO
	}
	return p.Profit().Mul(lib.ONE6).Div(p.Amount.Mul(lib.ONE12))
}

type PositionHistory struct {
	Id          int64       `json:"id"`
	Chain       int64       `json:"chain"`
	Index       int64       `json:"index"`
	Time        time.Time   `json:"time"`
	Shares      *lib.BigInt `json:"shares"`
	Borrow      *lib.BigInt `json:"borrow"`
	SharesValue *lib.BigInt `json:"shares_value"`
	BorrowValue *lib.BigInt `json:"borrow_value"`
	Life        *lib.BigInt `json:"life"`
	Amount      *lib.BigInt `json:"amount"`
	Price       *lib.BigInt `json:"price"`
}
