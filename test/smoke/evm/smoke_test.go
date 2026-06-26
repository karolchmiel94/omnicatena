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
	anvilRPCURL = "http://localhost:8545"
	anvilChainID = int64(31337)
	// Anvil account 0 — well-known test key, never use on mainnet.
	anvilPrivKeyHex = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
)

// fundAddress sends amount wei from Anvil's pre-funded account 0 to addr.
func fundAddress(ctx context.Context, t *testing.T, client *ethclient.Client, addr common.Address, amount *big.Int) {
	t.Helper()

	privKey, err := crypto.HexToECDSA(anvilPrivKeyHex)
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
	baseFee := head.BaseFee
	tip := big.NewInt(1e9)
	maxFee := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), tip)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(anvilChainID),
		Nonce:     nonce,
		GasTipCap: tip,
		GasFeeCap: maxFee,
		Gas:       21_000,
		To:        &addr,
		Value:     amount,
	})
	signer := types.NewLondonSigner(big.NewInt(anvilChainID))
	signed, err := types.SignTx(tx, signer, privKey)
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
		RPCURL:  anvilRPCURL,
		ChainID: anvilChainID,
		Chain:   domain.ChainEthereum,
	})
	if err != nil {
		t.Fatalf("evm.New: %v", err)
	}

	rawClient, err := ethclient.Dial(anvilRPCURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	ks := keystore.New()
	seed, err := ks.Create("smoke", []byte("test"))
	if err != nil {
		t.Fatalf("ks.Create: %v", err)
	}

	path := domain.DerivationPath("m/44'/60'/0'/0/0")
	acc, err := adapter.DeriveAccount(seed, path)
	if err != nil {
		t.Fatalf("DeriveAccount: %v", err)
	}
	t.Logf("address: %s", acc.Address)

	// Fund our address with 1 ETH from Anvil account 0.
	ourAddr := common.HexToAddress(string(acc.Address))
	oneEther := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1))
	fundAddress(ctx, t, rawClient, ourAddr, oneEther)
	time.Sleep(500 * time.Millisecond)

	bal, err := adapter.Balance(ctx, acc.Address, domain.Asset{})
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	t.Logf("balance: %s wei", bal.Base.String())
	if bal.Base.Cmp(big.NewInt(0)) == 0 {
		t.Fatal("expected non-zero balance after funding")
	}

	// Transfer 0.1 ETH to a fresh address.
	toKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	toAddr := domain.Address(crypto.PubkeyToAddress(toKey.PublicKey).Hex())
	t.Logf("recipient: %s", toAddr)

	sendAmount := new(big.Int).Mul(big.NewInt(1e17), big.NewInt(1)) // 0.1 ETH
	req := domain.TransferRequest{
		Network: domain.Network{Chain: domain.ChainEthereum},
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
	t.Logf("status=%s block=%d fee=%s wei", txInfo.Status, txInfo.BlockHeight, txInfo.Fee.Base.String())

	if txInfo.Status != domain.TxConfirmed {
		t.Errorf("expected TxConfirmed, got %s", txInfo.Status)
	}

	recipBal, err := adapter.Balance(ctx, toAddr, domain.Asset{})
	if err != nil {
		t.Fatalf("recipient balance: %v", err)
	}
	t.Logf("recipient balance: %s wei", recipBal.Base.String())
	if recipBal.Base.Cmp(sendAmount) != 0 {
		t.Errorf("expected %s wei at recipient, got %s", sendAmount, recipBal.Base.String())
	}
}
