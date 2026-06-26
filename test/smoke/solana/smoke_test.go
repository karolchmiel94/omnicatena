//go:build smoke

package smoke_test

import (
	"context"
	"fmt"
	"math/big"
	"os/exec"
	"strings"
	"testing"
	"time"

	solanago "github.com/gagliardetto/solana-go"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/solana"
	"github.com/karolchmiel94/omnicatena/internal/adapter/keystore"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	bip39 "github.com/tyler-smith/go-bip39"
)

func TestSmokeEndToEnd(t *testing.T) {
	ctx := context.Background()
	adapter := solana.New("http://localhost:8899")
	path := domain.DerivationPath("m/44'/501'/0'/0'")

	// Derive address from keystore-managed seed.
	ks := keystore.New()
	seed, err := ks.Create("smoke", []byte("test"))
	if err != nil {
		t.Fatalf("ks.Create: %v", err)
	}
	acc, err := adapter.DeriveAccount(seed, path)
	if err != nil {
		t.Fatalf("DeriveAccount: %v", err)
	}
	t.Logf("address: %s", acc.Address)

	// Airdrop 2 SOL.
	out, err := exec.Command("docker", "exec", "omnicatena-solana-1",
		"solana", "airdrop", "2", string(acc.Address), "--url", "http://localhost:8899",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("airdrop: %v\n%s", err, out)
	}
	t.Logf("airdrop: %s", strings.TrimSpace(string(out)))
	time.Sleep(6 * time.Second)

	// Balance.
	bal, err := adapter.Balance(ctx, acc.Address, domain.Asset{})
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	t.Logf("balance: %s lamports", bal.Base.String())
	if bal.Base.Cmp(big.NewInt(0)) == 0 {
		t.Fatal("expected non-zero balance after airdrop")
	}

	// Transfer 0.1 SOL to a fresh throwaway address.
	toKey := solanago.NewWallet()
	toAddr := domain.Address(toKey.PublicKey().String())
	t.Logf("recipient: %s", toAddr)

	_ = bip39.ErrInvalidMnemonic // ensure import used

	req := domain.TransferRequest{
		Network: domain.Network{Chain: domain.ChainSolana},
		From:    acc.Address,
		To:      toAddr,
		Amount: domain.Amount{
			Asset: domain.Asset{Symbol: "SOL", Decimals: 9, Native: true},
			Base:  big.NewInt(100_000_000), // 0.1 SOL
		},
		Speed: domain.SpeedStandard,
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

	time.Sleep(4 * time.Second)

	tx, err := adapter.GetTransaction(ctx, txHash)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	t.Logf("status=%s slot=%d fee=%s lamports", tx.Status, tx.BlockHeight, tx.Fee.Base.String())

	if tx.Status != domain.TxConfirmed {
		t.Errorf("expected TxConfirmed, got %s", tx.Status)
	}

	// Recipient should now have 0.1 SOL.
	recipBal, err := adapter.Balance(ctx, toAddr, domain.Asset{})
	if err != nil {
		t.Fatalf("recipient balance: %v", err)
	}
	fmt.Printf("recipient balance: %s lamports\n", recipBal.Base.String())
	if recipBal.Base.Cmp(big.NewInt(100_000_000)) != 0 {
		t.Errorf("expected 100000000 lamports at recipient, got %s", recipBal.Base.String())
	}
}
