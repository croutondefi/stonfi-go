package stonfi

import (
	"context"
	"math/big"

	"github.com/shopspring/decimal"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type RouterGasConstants struct {
	Swap             *big.Int
	ProvideLp        *big.Int
	SwapForward      *big.Int
	SwapTon          *big.Int
	ProvideLpForward *big.Int
}

type CreateSwapBodyParams struct {
	UserWalletAddress      *address.Address
	MinAskAmount           *big.Int
	AskJettonWalletAddress *address.Address
	ReferralAddress        *address.Address
}

type RouterRevision interface {
	GasConstants() RouterGasConstants
	CreateSwapBody(CreateSwapBodyParams) *cell.Cell
	CreateProvideLiquidityBody(
		routerWalletAddress *address.Address,
		minLpOut *decimal.Decimal,
	) *cell.Cell

	GetPoolAddress(context.Context, *address.Address, *address.Address) (*address.Address, error)

	ConstructPoolRevision(*address.Address) PoolRevision
	GetData(ctx context.Context) (*RouterData, error)
}

func NewRouterRevisionV1(api *ton.APIClient, addr *address.Address) RouterRevision {
	return &routerRevisionV1{
		api:  api,
		addr: addr,
	}
}

type routerRevisionV1 struct {
	api  *ton.APIClient
	addr *address.Address
}

func (r *routerRevisionV1) GasConstants() RouterGasConstants {
	return RouterGasConstants{
		Swap:             big.NewInt(300000000), //0.3 TON
		ProvideLp:        big.NewInt(300000000),
		SwapForward:      big.NewInt(265000000),
		SwapTon:          big.NewInt(240000000),
		ProvideLpForward: big.NewInt(265000000),
	}
}

func (r *routerRevisionV1) CreateSwapBody(params CreateSwapBodyParams) *cell.Cell {
	builder := cell.BeginCell().
		MustStoreUInt(OpCodeSwap, 32).
		MustStoreAddr(params.AskJettonWalletAddress).
		MustStoreCoins(params.MinAskAmount.Uint64()).
		MustStoreAddr(params.UserWalletAddress)

	if params.ReferralAddress != nil {
		builder.MustStoreUInt(1, 1)
		builder.MustStoreAddr(params.ReferralAddress)
	} else {
		builder.MustStoreUInt(0, 1)
	}

	return builder.EndCell()
}

func (r *routerRevisionV1) CreateProvideLiquidityBody(
	routerWalletAddress *address.Address,
	minLpOut *decimal.Decimal) *cell.Cell {

	builder := cell.BeginCell().
		MustStoreUInt(OpCodeProvideLiquidity, 32).
		MustStoreAddr(routerWalletAddress).
		MustStoreCoins(minLpOut.BigInt().Uint64())

	return builder.EndCell()
}

func (r *routerRevisionV1) GetPoolAddress(
	ctx context.Context,
	token0 *address.Address,
	token1 *address.Address,
) (*address.Address, error) {
	var cellA = cell.BeginCell().MustStoreAddr(token0).EndCell()
	var cellB = cell.BeginCell().MustStoreAddr(token1).EndCell()

	b, err := r.api.GetMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}

	res, err := r.api.RunGetMethod(ctx, b, r.addr, "get_pool_address", cellA.BeginParse(), cellB.BeginParse())
	if err != nil {
		return nil, err
	}

	resCell, err := res.Slice(0)
	if err != nil {
		return nil, err
	}

	poolAddr, err := resCell.LoadAddr()
	if err != nil {
		return nil, err
	}

	return poolAddr, err
}

func (r *routerRevisionV1) ConstructPoolRevision(addr *address.Address) PoolRevision {
	return NewPoolRevisionV1(r.api, addr)
}

func (r *routerRevisionV1) GetData(ctx context.Context) (*RouterData, error) {
	b, err := r.api.GetMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}

	res, err := r.api.RunGetMethod(ctx, b, r.addr, "get_router_data")
	if err != nil {
		return nil, err
	}

	rd := RouterData{}

	rd.IsLocked = (res.AsTuple()[0].(*big.Int).Int64() == 0)

	adminAddress := res.MustSlice(1)
	rd.AdminAddress = adminAddress.MustLoadAddr()
	rd.TempUpgrade = res.MustCell(2)
	rd.PoolCode = res.MustCell(3)
	rd.JettonLpWalletCode = res.MustCell(4)
	rd.LpAccountCode = res.MustCell(5)

	return &rd, nil
}
