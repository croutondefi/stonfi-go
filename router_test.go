package stonfi_test

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/croutondefi/stonfi-go"
	"github.com/stretchr/testify/assert"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

var api *ton.APIClient
var client *liteclient.ConnectionPool
var routerAddr *address.Address
var router stonfi.Router
var w *wallet.Wallet

const jetton0 = "EQCM3B12QK1e4yZSf8GtBRT0aLMNyEsBc_DhVfRRtOEffLez" //proxy ton
const jetton1 = "EQDnNof1yihA_JtXc70AZJCgw5_b1hIKkJGcdBXS5K-p3S-_"

func TestMain(m *testing.M) {
	client = liteclient.NewConnectionPool()

	err := client.AddConnectionsFromConfigUrl(context.Background(), "https://space.tonbetapp.com/utils/global.config.json")
	if err != nil {
		panic(err)
	}

	api = ton.NewAPIClient(client)

	routerAddr = address.MustParseAddr(stonfi.RouterRevisionV1Addr)
	routerRevV1 := stonfi.NewRouterRevisionV1(api, routerAddr)

	router = stonfi.NewRouter(api, routerAddr, routerRevV1)

	words := strings.Split(os.Getenv("seed"), " ")

	w, err = wallet.FromSeed(api, words, wallet.V4R2)
	if err != nil {
		panic(err)
	}

	code := m.Run()

	os.Exit(code)
}

func TestGetPoolData(t *testing.T) {
	ctx := client.StickyContext(context.Background())

	pool, err := router.GetPool(ctx, address.MustParseAddr(jetton0), address.MustParseAddr(jetton1))

	if err != nil {
		t.Errorf("Failed to get pool: %s", err)
		return
	}

	poolData, err := pool.GetData(ctx)
	if err != nil {
		t.Errorf("Failed to get pool data: %s", err)
		return
	}

	assert.NotNil(t, poolData.Reserve0)
	assert.NotNil(t, poolData.Reserve1)
}

func TestGetRouterData(t *testing.T) {
	ctx := client.StickyContext(context.Background())

	data, err := router.GetData(ctx)

	if err != nil {
		t.Errorf("Failed to get router data: %s", err)
		return
	}

	assert.NotNil(t, data.AdminAddress)
	assert.NotNil(t, data.JettonLpWalletCode)
	assert.NotNil(t, data.LpAccountCode)
	assert.NotNil(t, data.PoolCode)
}

func TestSwapTonToToken(t *testing.T) {
	ctx := client.StickyContext(context.Background())

	offerAmount := big.NewInt(300000000)

	data, err := router.BuildSwapProxyTonTxParams(ctx, stonfi.SwapProxyTonParams{
		UserWalletAddress: w.Address(),
		MinAskAmount:      big.NewInt(50000000000),
		OfferAmount:       offerAmount,
		AskJettonAddress:  address.MustParseAddr(jetton1),
		ProxyTonAddress:   address.MustParseAddr(jetton0),
		QueryId:           294082696817434,
	})

	if err != nil {
		t.Errorf("Failed to BuildSwapProxyTonTxParams: %s", err)
		return
	}

	tx, block, err := w.SendWaitTransaction(context.Background(), &wallet.Message{
		Mode: 3,
		InternalMessage: &tlb.InternalMessage{
			Bounce:  true,
			DstAddr: data.To,
			Amount:  tlb.FromNanoTON(data.Amount),
			Body:    data.Payload,
		},
	})

	fmt.Println(tx, block, err)

	t.FailNow()
}

func TestSwapTokenToTon(t *testing.T) {
	ctx := client.StickyContext(context.Background())

	offerAmount := big.NewInt(300000000000)

	data, err := router.BuildSwapJettonTxParams(ctx, stonfi.SwapJettonParams{
		UserWalletAddress:  w.Address(),
		MinAskAmount:       big.NewInt(100000000),
		OfferAmount:        offerAmount,
		AskJettonAddress:   address.MustParseAddr(jetton0),
		OfferJettonAddress: address.MustParseAddr(jetton1),
		QueryId:            294082696817435,
	})

	if err != nil {
		t.Errorf("Failed to BuildSwapProxyTonTxParams: %s", err)
		return
	}

	tx, block, err := w.SendWaitTransaction(context.Background(), &wallet.Message{
		Mode: 3,
		InternalMessage: &tlb.InternalMessage{
			Bounce:  true,
			DstAddr: data.To,
			Amount:  tlb.FromNanoTON(data.Amount),
			Body:    data.Payload,
		},
	})

	fmt.Println(tx, block, err)

	t.FailNow()
}
