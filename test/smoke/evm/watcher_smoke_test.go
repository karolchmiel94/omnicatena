//go:build smoke

package smoke_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/evm"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// TestSmokeWatcher: start the EVM watcher against real Anvil, send a real
// funded transfer to a watched address, confirm the matching TxEvent arrives
// on the channel with the right chain/type/address/hash/amount.
func TestSmokeWatcher(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	watcher, err := evm.NewWatcher(evm.Config{
		RPCURL:  anvilRPCURL,
		ChainID: anvilChainID,
		Chain:   domain.ChainEthereum,
	})
	if err != nil {
		t.Fatalf("evm.NewWatcher: %v", err)
	}

	toKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	watched := domain.Address(crypto.PubkeyToAddress(toKey.PublicKey).Hex())

	events, err := watcher.Watch(ctx, []domain.Address{watched})
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	rawClient, err := ethclient.Dial(anvilRPCURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	amount := big.NewInt(5e17) // 0.5 ETH
	fundAddress(ctx, t, rawClient, common.HexToAddress(string(watched)), amount)

	select {
	case evt, ok := <-events:
		if !ok {
			t.Fatal("event channel closed before delivering a match")
		}
		if evt.Chain != domain.ChainEthereum {
			t.Errorf("chain = %s, want %s", evt.Chain, domain.ChainEthereum)
		}
		if evt.Type != domain.EventInbound {
			t.Errorf("type = %s, want %s", evt.Type, domain.EventInbound)
		}
		if evt.Address != watched {
			t.Errorf("address = %s, want %s", evt.Address, watched)
		}
		if evt.Amount.Base.Cmp(amount) != 0 {
			t.Errorf("amount = %s, want %s", evt.Amount.Base, amount)
		}
		if evt.Status != domain.TxConfirmed {
			t.Errorf("status = %s, want %s", evt.Status, domain.TxConfirmed)
		}
		t.Logf("matched event: %+v", evt)
	case <-ctx.Done():
		t.Fatal("timed out waiting for watcher to report the funding tx")
	}
}
