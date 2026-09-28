// Command api wires adapters → app services and serves the HTTP transport.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/bitcoin"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/evm"
	solanadapter "github.com/karolchmiel94/omnicatena/internal/adapter/chain/solana"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/tron"
	"github.com/karolchmiel94/omnicatena/internal/adapter/events"
	"github.com/karolchmiel94/omnicatena/internal/adapter/events/kafka"
	"github.com/karolchmiel94/omnicatena/internal/adapter/keystore"
	"github.com/karolchmiel94/omnicatena/internal/adapter/repository"
	"github.com/karolchmiel94/omnicatena/internal/app"
	"github.com/karolchmiel94/omnicatena/internal/config"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/port"
	transport "github.com/karolchmiel94/omnicatena/internal/transport/http"
)

func main() {
	cfg := config.Load()

	dbPool, err := pgxpool.New(context.Background(), cfg.Database.URL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer dbPool.Close()
	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("db: ping: %v", err)
	}

	ethAdapter, err := evm.New(evm.Config{
		RPCURL:  cfg.Ethereum.RPCURL,
		ChainID: cfg.Ethereum.ChainID,
		Chain:   domain.ChainEthereum,
	})
	if err != nil {
		log.Fatalf("eth adapter: %v", err)
	}

	baseAdapter, err := evm.New(evm.Config{
		RPCURL:  cfg.Base.RPCURL,
		ChainID: cfg.Base.ChainID,
		Chain:   domain.ChainBase,
	})
	if err != nil {
		log.Fatalf("base adapter: %v", err)
	}

	btcAdapter, err := bitcoin.New(bitcoin.Config{
		Host:        cfg.Bitcoin.Host,
		User:        cfg.Bitcoin.User,
		Pass:        cfg.Bitcoin.Pass,
		ChainParams: &chaincfg.RegressionNetParams,
	})
	if err != nil {
		log.Fatalf("btc adapter: %v", err)
	}

	solAdapter := solanadapter.New(cfg.Solana.RPCURL)
	tronAdapter := tron.New(tron.Config{RPCURL: cfg.Tron.RPCURL})

	registry := chain.NewRegistry([]port.ChainAdapter{ethAdapter, baseAdapter, btcAdapter, solAdapter, tronAdapter})
	keys := keystore.NewPostgres(dbPool)
	repo := repository.NewPostgresWallet(dbPool)

	walletSvc := app.NewWalletService(registry, keys, repo)
	txSvc := app.NewTransactionService(registry, keys, repo)

	// Block-scanning watchers — EVM only for now (Ethereum + Base); Bitcoin,
	// Solana, TRON watchers follow the same pattern as a later addition.
	cursors := repository.NewPostgresCursor(dbPool)
	ethWatcher, err := evm.NewWatcher(evm.Config{RPCURL: cfg.Ethereum.RPCURL, ChainID: cfg.Ethereum.ChainID, Chain: domain.ChainEthereum})
	if err != nil {
		log.Fatalf("eth watcher: %v", err)
	}
	ethWatcher.SetCursorStore(cursors)
	baseWatcher, err := evm.NewWatcher(evm.Config{RPCURL: cfg.Base.RPCURL, ChainID: cfg.Base.ChainID, Chain: domain.ChainBase})
	if err != nil {
		log.Fatalf("base watcher: %v", err)
	}
	baseWatcher.SetCursorStore(cursors)
	registry.RegisterWatcher(ethWatcher)
	registry.RegisterWatcher(baseWatcher)

	var publisher port.TxEventPublisher = events.Noop{}
	if cfg.Kafka.Enabled {
		kafkaPub := kafka.New(cfg.Kafka.Brokers, cfg.Kafka.Topic)
		defer kafkaPub.Close()
		publisher = kafkaPub
	}
	txRepo := repository.NewPostgresTransaction(dbPool)
	monitorSvc := app.NewMonitorService(registry, publisher, txRepo)
	startMonitoring(monitorSvc, repo)

	h := transport.NewHandler(walletSvc, txSvc)
	log.Println("api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", h.Router()))
}
