// Command api wires adapters → app services and serves the HTTP transport.
package main

import (
	"log"
	"net/http"

	"github.com/karolchmiel94/omnicatena/internal/adapter/chain"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/evm"
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

	ethAdapter, err := evm.New(evm.Config{
		RPCURL:  cfg.Ethereum.RPCURL,
		ChainID: cfg.Ethereum.ChainID,
		Chain:   domain.ChainEthereum,
	})
	if err != nil {
		log.Fatalf("eth adapter: %v", err)
	}

	registry := chain.NewRegistry([]port.ChainAdapter{ethAdapter})
	keys := keystore.New()
	repo := repository.NewInMemoryWallet()

	walletSvc := app.NewWalletService(registry, keys, repo)
	txSvc := app.NewTransactionService(registry, keys, repo)

	h := transport.NewHandler(walletSvc, txSvc)
	log.Println("api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", h.Router()))
}
