//go:build smoke

package smoke_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/evm"
	"github.com/karolchmiel94/omnicatena/internal/adapter/keystore"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

const (
	baseRPCURL  = "http://localhost:8546"
	baseChainID = int64(8453)
	// Anvil account 0 — well-known test key, never use on mainnet.
	baseAnvilPrivKey = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
)

func fundBaseAddress(ctx context.Context, t *testing.T, client *ethclient.Client, addr common.Address, amount *big.Int) {
	t.Helper()
	privKey, err := crypto.HexToECDSA(baseAnvilPrivKey)
	if err != nil {
		t.Fatalf("parse anvil key: %v", err)
	}
	from := crypto.PubkeyToAddress(privKey.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		t.Fatalf("nonce: %v", err)
	}
	head, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		t.Fatalf("header: %v", err)
	}
	tip := big.NewInt(1e9)
	maxFee := new(big.Int).Add(new(big.Int).Mul(head.BaseFee, big.NewInt(2)), tip)
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(baseChainID),
		Nonce:     nonce,
		GasTipCap: tip,
		GasFeeCap: maxFee,
		Gas:       21_000,
		To:        &addr,
		Value:     amount,
	})
	signed, err := types.SignTx(tx, types.NewLondonSigner(big.NewInt(baseChainID)), privKey)
	if err != nil {
		t.Fatalf("sign fund tx: %v", err)
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		t.Fatalf("send fund tx: %v", err)
	}
}

func TestSmokeEndToEnd(t *testing.T) {
	ctx := context.Background()

	adapter, err := evm.New(evm.Config{
		RPCURL:  baseRPCURL,
		ChainID: baseChainID,
		Chain:   domain.ChainBase,
	})
	if err != nil {
		t.Fatalf("evm.New: %v", err)
	}
	rawClient, err := ethclient.Dial(baseRPCURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	ks := keystore.New()
	seed, err := ks.Create("smoke", []byte("test"))
	if err != nil {
		t.Fatalf("ks.Create: %v", err)
	}

	acc, err := adapter.DeriveAccount(seed, "m/44'/60'/0'/0/0")
	if err != nil {
		t.Fatalf("DeriveAccount: %v", err)
	}
	t.Logf("address: %s", acc.Address)

	fundBaseAddress(ctx, t, rawClient, common.HexToAddress(string(acc.Address)), new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1)))
	time.Sleep(500 * time.Millisecond)

	bal, err := adapter.Balance(ctx, acc.Address, domain.Asset{})
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	t.Logf("balance: %s wei", bal.Base)
	if bal.Base.Cmp(big.NewInt(0)) == 0 {
		t.Fatal("expected non-zero balance after funding")
	}

	toKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	toAddr := domain.Address(crypto.PubkeyToAddress(toKey.PublicKey).Hex())
	t.Logf("recipient: %s", toAddr)

	sendAmount := big.NewInt(1e17) // 0.1 ETH
	req := domain.TransferRequest{
		Network: domain.Network{Chain: domain.ChainBase},
		From:    acc.Address,
		To:      toAddr,
		Amount:  domain.Amount{Asset: domain.Asset{Symbol: "ETH", Decimals: 18, Native: true}, Base: sendAmount},
		Speed:   domain.SpeedStandard,
	}

	unsigned, err := adapter.BuildTransfer(ctx, req)
	if err != nil {
		t.Fatalf("BuildTransfer: %v", err)
	}
	unsigned.Account = acc

	signer, err := ks.Signer("smoke", []byte("test"))
	if err != nil {
		t.Fatalf("Signer: %v", err)
	}

	signed, err := adapter.Sign(ctx, unsigned, signer)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	t.Logf("signed hash: %s", signed.Hash)

	txHash, err := adapter.Broadcast(ctx, signed)
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	t.Logf("broadcast: %s", txHash)
	time.Sleep(2 * time.Second)

	txInfo, err := adapter.GetTransaction(ctx, txHash)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	t.Logf("status=%s block=%d fee=%s wei", txInfo.Status, txInfo.BlockHeight, txInfo.Fee.Base)
	if txInfo.Status != domain.TxConfirmed {
		t.Errorf("expected TxConfirmed, got %s", txInfo.Status)
	}

	recipBal, err := adapter.Balance(ctx, toAddr, domain.Asset{})
	if err != nil {
		t.Fatalf("recipient balance: %v", err)
	}
	t.Logf("recipient balance: %s wei", recipBal.Base)
	if recipBal.Base.Cmp(sendAmount) != 0 {
		t.Errorf("expected %s wei, got %s", sendAmount, recipBal.Base)
	}
}
