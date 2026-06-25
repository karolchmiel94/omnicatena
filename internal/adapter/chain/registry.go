package chain

import (
	"fmt"

	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/port"
)

type Registry struct {
	adapters map[domain.ChainID]port.ChainAdapter
	watchers map[domain.ChainID]port.ChainWatcher
}

func NewRegistry(adapters []port.ChainAdapter) *Registry {
	r := &Registry{
		adapters: make(map[domain.ChainID]port.ChainAdapter, len(adapters)),
		watchers: make(map[domain.ChainID]port.ChainWatcher),
	}
	for _, a := range adapters {
		r.adapters[a.Chain()] = a
	}
	return r
}

func (r *Registry) Adapter(chain domain.ChainID) (port.ChainAdapter, error) {
	a, ok := r.adapters[chain]
	if !ok {
		return nil, fmt.Errorf("chain: no adapter for %s", chain)
	}
	return a, nil
}

func (r *Registry) Watcher(chain domain.ChainID) (port.ChainWatcher, error) {
	w, ok := r.watchers[chain]
	if !ok {
		return nil, fmt.Errorf("chain: no watcher for %s", chain)
	}
	return w, nil
}

func (r *Registry) Supported() []domain.ChainID {
	out := make([]domain.ChainID, 0, len(r.adapters))
	for id := range r.adapters {
		out = append(out, id)
	}
	return out
}
