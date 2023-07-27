package stonfi

import (
	"context"
	"math/big"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
)

type PoolRevision interface {
	GasConstants() PoolGasConstants
	GetData(context.Context) (*PoolData, error)

	// CreateCollectFeesBody(QueryId) *cell.Cell
	// CreateBurnBody(decimal.Decimal, *address.Address, QueryId) *cell.Cell
	// GetExpectedOutputs(decimal.Decimal, *address.Address) ExpectedOutputsData
	// GetExpectedTokens(decimal.Decimal, decimal.Decimal) *bit.Int
	// GetExpectedLiquidity(decimal.Decimal) PoolAmountsData
	// GetLpAccountAddress(*address.Address) *address.Address
	// ConstructLpAccountRevision(pool: Pool): LpAccountRevision;
}

func NewPoolRevisionV1(api *ton.APIClient, addr *address.Address) PoolRevision {
	return &poolRevisionV1{
		api:  api,
		addr: addr,
	}
}

type poolRevisionV1 struct {
	api  *ton.APIClient
	addr *address.Address
}

func (p *poolRevisionV1) GasConstants() PoolGasConstants {
	return PoolGasConstants{
		CollectFees: big.NewInt(1100000000),
		Burn:        big.NewInt(500000000),
	}
}

func (p *poolRevisionV1) GetData(ctx context.Context) (*PoolData, error) {
	b, err := p.api.GetMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}

	res, err := p.api.RunGetMethod(ctx, b, p.addr, "get_pool_data")
	if err != nil {
		return nil, err
	}

	pd := PoolData{}

	pd.Reserve0 = res.AsTuple()[0].(*big.Int)
	pd.Reserve1 = res.AsTuple()[1].(*big.Int)

	token0WalletCell := res.MustSlice(2)
	token1WalletCell := res.MustSlice(3)
	pd.Token0WalletAddress = token0WalletCell.MustLoadAddr()
	pd.Token1WalletAddress = token1WalletCell.MustLoadAddr()

	pd.LpFee = res.AsTuple()[4].(*big.Int)
	pd.ProtocolFee = res.AsTuple()[5].(*big.Int)
	pd.RefFee = res.AsTuple()[6].(*big.Int)

	protocolFeeAddressCell := res.MustSlice(7)
	pd.ProtocolFeeAddress = protocolFeeAddressCell.MustLoadAddr()

	pd.CollectedToken0ProtocolFee = res.AsTuple()[8].(*big.Int)
	pd.CollectedToken1ProtocolFee = res.AsTuple()[9].(*big.Int)

	return &pd, nil
}
