package main

import (
	"context"
	"log"

	"github.com/karolchmiel94/omnicatena/internal/app"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/port"
)

// startMonitoring watches whatever wallet addresses exist in the repo at
// startup. With persistence this now reflects every wallet created before
// this boot; it still doesn't refresh for wallets created after — that needs
// either a periodic reload or the repo notifying on writes, neither of which
// is in scope here.
func startMonitoring(svc *app.MonitorService, repo port.WalletRepository) {
	wallets, err := repo.List(context.Background())
	if err != nil {
		log.Printf("monitor: list wallets: %v", err)
		return
	}
	for _, chain := range []domain.ChainID{domain.ChainEthereum, domain.ChainBase} {
		addrs := make(map[domain.Address]string)
		for _, w := range wallets {
			if acc, ok := w.Account(chain); ok {
				addrs[acc.Address] = w.ID
			}
		}
		go func(chain domain.ChainID, addrs map[domain.Address]string) {
			if err := svc.Watch(context.Background(), chain, addrs); err != nil {
				log.Printf("monitor: watch %s: %v", chain, err)
			}
		}(chain, addrs)
	}
}
