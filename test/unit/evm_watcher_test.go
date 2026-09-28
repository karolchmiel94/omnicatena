package unit_test

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/evm"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

const watcherTestChainID = 31337

var watcherTestSigner = types.NewLondonSigner(big.NewInt(watcherTestChainID))

func mustKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return key
}

func signTestTx(t *testing.T, key *ecdsa.PrivateKey, to *common.Address, nonce uint64) *types.Transaction {
	t.Helper()
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(watcherTestChainID),
		Nonce:     nonce,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		Gas:       21_000,
		To:        to,
		Value:     big.NewInt(1_000_000),
	})
	signed, err := types.SignTx(tx, watcherTestSigner, key)
	if err != nil {
		t.Fatalf("SignTx: %v", err)
	}
	return signed
}

func TestMatchTransactions(t *testing.T) {
	keyA, keyB, keyStranger := mustKey(t), mustKey(t), mustKey(t)
	addrA := crypto.PubkeyToAddress(keyA.PublicKey)
	addrB := crypto.PubkeyToAddress(keyB.PublicKey)
	addrStranger := crypto.PubkeyToAddress(keyStranger.PublicKey)

	watched := map[common.Address]domain.Address{
		addrA: domain.Address(addrA.Hex()),
		addrB: domain.Address(addrB.Hex()),
	}

	outbound := signTestTx(t, keyA, &addrStranger, 0)          // watched -> unwatched
	inbound := signTestTx(t, keyStranger, &addrB, 0)           // unwatched -> watched
	both := signTestTx(t, keyA, &addrB, 1)                     // watched -> watched
	irrelevant := signTestTx(t, keyStranger, &addrStranger, 1) // unwatched -> unwatched
	contractCreation := signTestTx(t, keyA, nil, 2)            // watched -> nil (To() == nil)

	txs := []*types.Transaction{outbound, inbound, both, irrelevant, contractCreation}
	const blockNumber = uint64(42)
	const blockTime = uint64(1_700_000_000)

	events := evm.MatchTransactions(domain.ChainEthereum, watcherTestSigner, txs, blockNumber, blockTime, watched)

	want := map[string]domain.TxEventType{
		outbound.Hash().Hex():         domain.EventOutbound,
		inbound.Hash().Hex():          domain.EventInbound,
		contractCreation.Hash().Hex(): domain.EventOutbound,
	}
	// "both" produces two events (outbound for A, inbound for B) — checked separately below.

	gotByHash := make(map[string][]domain.TxEvent)
	for _, e := range events {
		gotByHash[e.Hash] = append(gotByHash[e.Hash], e)
	}

	if _, ok := gotByHash[irrelevant.Hash().Hex()]; ok {
		t.Errorf("irrelevant tx should produce no events, got %v", gotByHash[irrelevant.Hash().Hex()])
	}

	for hash, wantType := range want {
		got := gotByHash[hash]
		if len(got) != 1 {
			t.Fatalf("tx %s: got %d events, want 1: %+v", hash, len(got), got)
		}
		if got[0].Type != wantType {
			t.Errorf("tx %s: type = %s, want %s", hash, got[0].Type, wantType)
		}
		if got[0].Chain != domain.ChainEthereum {
			t.Errorf("tx %s: chain = %s, want %s", hash, got[0].Chain, domain.ChainEthereum)
		}
		if got[0].BlockHeight != blockNumber {
			t.Errorf("tx %s: block height = %d, want %d", hash, got[0].BlockHeight, blockNumber)
		}
		if !got[0].Timestamp.Equal(got[0].Timestamp) || got[0].Timestamp.Unix() != int64(blockTime) {
			t.Errorf("tx %s: timestamp = %v, want unix %d", hash, got[0].Timestamp, blockTime)
		}
	}

	// outbound (A -> stranger): counterparty is the stranger.
	if got := gotByHash[outbound.Hash().Hex()][0]; got.Counterparty != domain.Address(addrStranger.Hex()) {
		t.Errorf("outbound counterparty = %s, want %s", got.Counterparty, addrStranger.Hex())
	}
	// inbound (stranger -> B): counterparty is the stranger.
	if got := gotByHash[inbound.Hash().Hex()][0]; got.Counterparty != domain.Address(addrStranger.Hex()) {
		t.Errorf("inbound counterparty = %s, want %s", got.Counterparty, addrStranger.Hex())
	}
	// contract creation (A -> nil): no counterparty.
	if got := gotByHash[contractCreation.Hash().Hex()][0]; got.Counterparty != "" {
		t.Errorf("contract creation counterparty = %q, want empty", got.Counterparty)
	}

	bothEvents := gotByHash[both.Hash().Hex()]
	if len(bothEvents) != 2 {
		t.Fatalf("watched->watched tx: got %d events, want 2: %+v", len(bothEvents), bothEvents)
	}
	seenTypes := map[domain.TxEventType]bool{}
	for _, e := range bothEvents {
		seenTypes[e.Type] = true
	}
	if !seenTypes[domain.EventOutbound] || !seenTypes[domain.EventInbound] {
		t.Errorf("watched->watched tx: want both outbound and inbound, got %+v", bothEvents)
	}
	for _, e := range bothEvents {
		switch e.Type {
		case domain.EventOutbound: // A's side: counterparty is B
			if e.Address != domain.Address(addrA.Hex()) || e.Counterparty != domain.Address(addrB.Hex()) {
				t.Errorf("watched->watched outbound: address=%s counterparty=%s, want %s/%s", e.Address, e.Counterparty, addrA.Hex(), addrB.Hex())
			}
		case domain.EventInbound: // B's side: counterparty is A
			if e.Address != domain.Address(addrB.Hex()) || e.Counterparty != domain.Address(addrA.Hex()) {
				t.Errorf("watched->watched inbound: address=%s counterparty=%s, want %s/%s", e.Address, e.Counterparty, addrB.Hex(), addrA.Hex())
			}
		}
	}
}

func TestMatchTransactions_NoWatchedAddresses(t *testing.T) {
	key := mustKey(t)
	to := crypto.PubkeyToAddress(mustKey(t).PublicKey)
	tx := signTestTx(t, key, &to, 0)

	events := evm.MatchTransactions(domain.ChainEthereum, watcherTestSigner, []*types.Transaction{tx}, 0, 0, map[common.Address]domain.Address{})
	if len(events) != 0 {
		t.Errorf("empty watch set: got %d events, want 0", len(events))
	}
}
