package stonfi

import (
	"context"
	"math/big"

	"github.com/xssnick/tonutils-go/address"
)

type PoolData struct {
	Reserve0                   *big.Int
	Reserve1                   *big.Int
	Token0WalletAddress        *address.Address
	Token1WalletAddress        *address.Address
	LpFee                      *big.Int
	ProtocolFee                *big.Int
	RefFee                     *big.Int
	ProtocolFeeAddress         *address.Address
	CollectedToken0ProtocolFee *big.Int
	CollectedToken1ProtocolFee *big.Int
}

type PoolGasConstants struct {
	CollectFees *big.Int
	Burn        *big.Int
}

type Pool interface {
	GasConstants() PoolGasConstants
	GetData(context.Context) (*PoolData, error)
}
