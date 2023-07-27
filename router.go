package stonfi

import (
	"context"
	"fmt"
	"math/big"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type RouterData struct {
	IsLocked           bool
	AdminAddress       *address.Address
	TempUpgrade        *cell.Cell
	PoolCode           *cell.Cell
	JettonLpWalletCode *cell.Cell
	LpAccountCode      *cell.Cell
}

type MessageData struct {
	To      *address.Address
	Payload *cell.Cell
	Amount  *big.Int
}

type SwapProxyTonParams struct {
	UserWalletAddress *address.Address
	MinAskAmount      *big.Int
	AskJettonAddress  *address.Address
	ReferralAddress   *address.Address
	ProxyTonAddress   *address.Address
	OfferAmount       *big.Int
	ForwardGasAmount  *big.Int
	QueryId           uint64
}

type SwapJettonParams struct {
	UserWalletAddress  *address.Address
	MinAskAmount       *big.Int
	AskJettonAddress   *address.Address
	ReferralAddress    *address.Address
	OfferJettonAddress *address.Address
	OfferAmount        *big.Int
	ForwardGasAmount   *big.Int
	QueryId            uint64
}

type router struct {
	addr     *address.Address
	api      *ton.APIClient
	revision RouterRevision
}

func NewRouter(api *ton.APIClient, addr *address.Address, rev RouterRevision) Router {
	return &router{
		api:      api,
		addr:     addr,
		revision: rev,
	}
}

type Router interface {
	CreateSwapBody(CreateSwapBodyParams) *cell.Cell
	BuildSwapProxyTonTxParams(context.Context, SwapProxyTonParams) (*MessageData, error)
	BuildSwapJettonTxParams(context.Context, SwapJettonParams) (*MessageData, error)
	GetPoolAddress(context.Context, *address.Address, *address.Address) (*address.Address, error)
	GetData(context.Context) (*RouterData, error)
	GetPool(context.Context, *address.Address, *address.Address) (Pool, error)
}

func (r *router) GetPool(ctx context.Context, token0, token1 *address.Address) (Pool, error) {
	token0Client := jetton.NewJettonMasterClient(r.api, token0)

	token0Wallet, err := token0Client.GetJettonWallet(ctx, r.addr)
	if err != nil {
		return nil, err
	}

	token1Client := jetton.NewJettonMasterClient(r.api, token1)

	token1Wallet, err := token1Client.GetJettonWallet(ctx, r.addr)
	if err != nil {
		return nil, err
	}

	poolAddr, err := r.revision.GetPoolAddress(ctx, token0Wallet.Address(), token1Wallet.Address())
	if err != nil {
		return nil, fmt.Errorf("get pool address: %s", err)
	}

	return r.revision.ConstructPoolRevision(poolAddr), nil
}

func (r *router) CreateSwapBody(params CreateSwapBodyParams) *cell.Cell {
	return r.revision.CreateSwapBody(params)
}

func (r *router) GetPoolAddress(ctx context.Context, a0 *address.Address, a1 *address.Address) (*address.Address, error) {
	return r.revision.GetPoolAddress(ctx, a0, a1)
}

func (r *router) GetData(ctx context.Context) (*RouterData, error) {
	return r.revision.GetData(ctx)
}

func (r *router) BuildSwapJettonTxParams(ctx context.Context, params SwapJettonParams) (*MessageData, error) {
	askJetton := jetton.NewJettonMasterClient(r.api, params.AskJettonAddress)

	askJettonWallet, err := askJetton.GetJettonWallet(ctx, r.addr)
	if err != nil {
		return nil, err
	}

	offerJetton := jetton.NewJettonMasterClient(r.api, params.OfferJettonAddress)

	offerJettonWallet, err := offerJetton.GetJettonWallet(ctx, params.UserWalletAddress)
	if err != nil {
		return nil, err
	}

	forwardPayload := r.CreateSwapBody(CreateSwapBodyParams{
		UserWalletAddress:      params.UserWalletAddress,
		MinAskAmount:           params.MinAskAmount,
		AskJettonWalletAddress: askJettonWallet.Address(),
		ReferralAddress:        params.ReferralAddress,
	})

	var forwardTonAmount = r.revision.GasConstants().SwapForward

	if params.ForwardGasAmount != nil {
		forwardTonAmount = params.ForwardGasAmount
	}

	payload := createJettonTransferMessage(
		params.QueryId,
		params.OfferAmount,
		r.addr,
		nil,
		nil,
		forwardTonAmount,
		forwardPayload,
	)

	return &MessageData{
		To:      offerJettonWallet.Address(),
		Payload: payload,
		Amount:  r.revision.GasConstants().Swap,
	}, nil
}

func (r *router) BuildSwapProxyTonTxParams(ctx context.Context, params SwapProxyTonParams) (*MessageData, error) {
	askJetton := jetton.NewJettonMasterClient(r.api, params.AskJettonAddress)

	askJettonWallet, err := askJetton.GetJettonWallet(ctx, r.addr)
	if err != nil {
		return nil, err
	}

	proxyTon := jetton.NewJettonMasterClient(r.api, params.ProxyTonAddress)

	proxyTonWallet, err := proxyTon.GetJettonWallet(ctx, r.addr)
	if err != nil {
		return nil, err
	}

	forwardPayload := r.CreateSwapBody(CreateSwapBodyParams{
		UserWalletAddress:      params.UserWalletAddress,
		MinAskAmount:           params.MinAskAmount,
		AskJettonWalletAddress: askJettonWallet.Address(),
		ReferralAddress:        params.ReferralAddress,
	})

	var forwardTonAmount = r.revision.GasConstants().SwapTon

	if params.ForwardGasAmount != nil {
		forwardTonAmount = params.ForwardGasAmount
	}

	payload := createJettonTransferMessage(
		params.QueryId,
		params.OfferAmount,
		r.addr,
		nil,
		nil,
		forwardTonAmount,
		forwardPayload,
	)

	return &MessageData{
		To:      proxyTonWallet.Address(),
		Payload: payload,
		Amount:  big.NewInt(0).Add(r.revision.GasConstants().SwapTon, params.OfferAmount),
	}, nil
}

func createJettonTransferMessage(
	queryId uint64,
	amount *big.Int,
	destination *address.Address,
	responseDestination *address.Address,
	customPayload *cell.Cell,
	forwardTonAmount *big.Int,
	forwardPayload *cell.Cell,
) *cell.Cell {
	builder := cell.BeginCell().
		MustStoreUInt(OpCodeRequestTransfer, 32).
		MustStoreUInt(queryId, 64).
		MustStoreCoins(amount.Uint64()).
		MustStoreAddr(destination).
		MustStoreAddr(responseDestination)

	if customPayload != nil {
		builder.MustStoreRef(customPayload).
			MustStoreBoolBit(true)
	} else {
		builder.MustStoreBoolBit(false)
	}

	builder.MustStoreCoins(forwardTonAmount.Uint64())

	if forwardPayload != nil {
		builder.MustStoreRef(forwardPayload).
			MustStoreBoolBit(true)
	} else {
		builder.MustStoreBoolBit(false)
	}
	return builder.EndCell()
}
