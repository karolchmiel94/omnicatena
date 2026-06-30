// Command omni wires the same app services as the API behind a CLI.
package main

import (
	"log"
	"os"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/bitcoin"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/evm"
	solanadapter "github.com/karolchmiel94/omnicatena/internal/adapter/chain/solana"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/tron"
	"github.com/karolchmiel94/omnicatena/internal/adapter/keystore"
	"github.com/karolchmiel94/omnicatena/internal/adapter/repository"
	"github.com/karolchmiel94/omnicatena/internal/app"
	"github.com/karolchmiel94/omnicatena/internal/config"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/port"
	"github.com/karolchmiel94/omnicatena/internal/transport/cli"
)

func main() {
	cfg := config.Load()

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
	keys := keystore.New()
	repo := repository.NewInMemoryWallet()

	walletSvc := app.NewWalletService(registry, keys, repo)
	txSvc := app.NewTransactionService(registry, keys, repo)

	if err := cli.New(walletSvc, txSvc).Run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
