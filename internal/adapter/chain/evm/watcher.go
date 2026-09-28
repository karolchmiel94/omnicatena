package evm

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

const watcherPollInterval = 2 * time.Second

// Watcher implements port.ChainWatcher for EVM chains (Ethereum, Base — same
// code, different config, ADR-0007): polls for new blocks and scans every
// transaction in each one against the watched address set. No persistence —
// it starts from the chain tip at Watch() time, not from a saved cursor.
type Watcher struct {
	cfg    Config
	client *ethclient.Client
	signer types.Signer
}

func NewWatcher(cfg Config) (*Watcher, error) {
	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("evm: watcher dial %s: %w", cfg.RPCURL, err)
	}
	return &Watcher{
		cfg:    cfg,
		client: client,
		signer: types.NewLondonSigner(big.NewInt(cfg.ChainID)),
	}, nil
}

func (w *Watcher) Chain() domain.ChainID { return w.cfg.Chain }

func (w *Watcher) Watch(ctx context.Context, addrs []domain.Address) (<-chan domain.TxEvent, error) {
	tip, err := w.client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("evm: watcher latest block: %w", err)
	}

	watched := make(map[common.Address]domain.Address, len(addrs))
	for _, a := range addrs {
		watched[common.HexToAddress(string(a))] = a
	}

	out := make(chan domain.TxEvent)
	go w.poll(ctx, tip, watched, out)
	return out, nil
}

// poll scans blocks (tip+1, tip+2, ...) on a fixed interval. A per-block RPC
// error is logged and retried on the next tick rather than aborting the watch.
func (w *Watcher) poll(ctx context.Context, from uint64, watched map[common.Address]domain.Address, out chan<- domain.TxEvent) {
	defer close(out)
	ticker := time.NewTicker(watcherPollInterval)
	defer ticker.Stop()

	next := from + 1
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		tip, err := w.client.BlockNumber(ctx)
		if err != nil {
			log.Printf("evm watcher (%s): block number: %v", w.cfg.Chain, err)
			continue
		}
		for n := next; n <= tip; n++ {
			block, err := w.client.BlockByNumber(ctx, new(big.Int).SetUint64(n))
			if err != nil {
				log.Printf("evm watcher (%s): block %d: %v", w.cfg.Chain, n, err)
				break // retry this block from the next tick
			}
			for _, evt := range MatchTransactions(w.cfg.Chain, w.signer, block.Transactions(), block.Time(), watched) {
				select {
				case out <- evt:
				case <-ctx.Done():
					return
				}
			}
			next = n + 1
		}
	}
}

// MatchTransactions parses a block's transactions and returns a TxEvent for
// every watched address involved, as sender and/or recipient — both if a
// transfer is between two watched addresses. Pure aside from signature
// recovery (no RPC), so it's unit-testable without a live node.
func MatchTransactions(chain domain.ChainID, signer types.Signer, txs []*types.Transaction, blockTime uint64, watched map[common.Address]domain.Address) []domain.TxEvent {
	var events []domain.TxEvent
	ts := time.Unix(int64(blockTime), 0).UTC()

	for _, tx := range txs {
		if from, err := types.Sender(signer, tx); err == nil {
			if addr, ok := watched[from]; ok {
				events = append(events, txEvent(domain.EventOutbound, chain, addr, tx, ts))
			}
		}
		if to := tx.To(); to != nil {
			if addr, ok := watched[*to]; ok {
				events = append(events, txEvent(domain.EventInbound, chain, addr, tx, ts))
			}
		}
	}
	return events
}

func txEvent(typ domain.TxEventType, chain domain.ChainID, addr domain.Address, tx *types.Transaction, ts time.Time) domain.TxEvent {
	return domain.TxEvent{
		Type:      typ,
		Chain:     chain,
		Address:   addr,
		Hash:      tx.Hash().Hex(),
		Amount:    domain.Amount{Asset: nativeETH, Base: tx.Value()},
		Status:    domain.TxConfirmed,
		Timestamp: ts,
	}
}
