//go:build smoke

package smoke_test

import (
	"context"
	"math/big"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/bitcoin"
	"github.com/karolchmiel94/omnicatena/internal/adapter/keystore"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

const (
	btcRPCHost = "localhost:18443"
	btcUser    = "omni"
	btcPass    = "omni"
	btcContainer = "omnicatena-bitcoind-1"
)

func btcCLI(t *testing.T, args ...string) string {
	t.Helper()
	base := []string{"exec", btcContainer, "bitcoin-cli",
		"-regtest", "-rpcuser=" + btcUser, "-rpcpassword=" + btcPass}
	out, err := exec.Command("docker", append(base, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("bitcoin-cli %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestSmokeEndToEnd(t *testing.T) {
	ctx := context.Background()

	adapter, err := bitcoin.New(bitcoin.Config{
		Host:        btcRPCHost,
		User:        btcUser,
		Pass:        btcPass,
		ChainParams: &chaincfg.RegressionNetParams,
	})
	if err != nil {
		t.Fatalf("bitcoin.New: %v", err)
	}

	ks := keystore.New()
	seed, err := ks.Create("smoke", []byte("test"))
	if err != nil {
		t.Fatalf("ks.Create: %v", err)
	}

	path := domain.DerivationPath("m/44'/0'/0'/0/0")
	acc, err := adapter.DeriveAccount(seed, path)
	if err != nil {
		t.Fatalf("DeriveAccount: %v", err)
	}
	t.Logf("address: %s", acc.Address)

	// Mine 101 blocks to our address so the coinbase is spendable.
	out := btcCLI(t, "generatetoaddress", "101", string(acc.Address))
	t.Logf("mined 101 blocks, last hash: %s", out[len(out)-64:])

	time.Sleep(2 * time.Second)

	bal, err := adapter.Balance(ctx, acc.Address, domain.Asset{})
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	t.Logf("balance: %s sat", bal.Base.String())
	if bal.Base.Cmp(big.NewInt(0)) == 0 {
		t.Fatal("expected non-zero balance after mining")
	}

	// Derive recipient address from a different index.
	recipAcc, err := adapter.DeriveAccount(seed, "m/44'/0'/0'/0/1")
	if err != nil {
		t.Fatalf("DeriveAccount recipient: %v", err)
	}
	t.Logf("recipient: %s", recipAcc.Address)

	const oneBTC = int64(100_000_000)
	req := domain.TransferRequest{
		Network: domain.Network{Chain: domain.ChainBitcoin},
		From:    acc.Address,
		To:      recipAcc.Address,
		Amount:  domain.Amount{Asset: domain.Asset{Symbol: "BTC", Decimals: 8, Native: true}, Base: big.NewInt(oneBTC)},
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

	// Mine 1 block to confirm the transaction.
	btcCLI(t, "generatetoaddress", "1", string(acc.Address))
	time.Sleep(2 * time.Second)

	txInfo, err := adapter.GetTransaction(ctx, txHash)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	t.Logf("status=%s confirmations=%d", txInfo.Status, txInfo.Confirmations)

	if txInfo.Status != domain.TxConfirmed {
		t.Errorf("expected TxConfirmed, got %s", txInfo.Status)
	}

	recipBal, err := adapter.Balance(ctx, recipAcc.Address, domain.Asset{})
	if err != nil {
		t.Fatalf("recipient balance: %v", err)
	}
	t.Logf("recipient balance: %s sat", recipBal.Base.String())
	if recipBal.Base.Cmp(big.NewInt(oneBTC)) != 0 {
		t.Errorf("expected %d sat at recipient, got %s", oneBTC, recipBal.Base.String())
	}
}
