package http

import (
	"encoding/json"
	"math/big"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/karolchmiel94/omnicatena/internal/app"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

type Handler struct {
	wallets      *app.WalletService
	transactions *app.TransactionService
}

func NewHandler(w *app.WalletService, t *app.TransactionService) *Handler {
	return &Handler{wallets: w, transactions: t}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/wallets", h.createWallet)
	r.Get("/wallets", h.listWallets)
	r.Get("/wallets/{id}", h.getWallet)
	r.Get("/wallets/{id}/balance/{chain}", h.walletBalance)

	r.Post("/transactions", h.transfer)
	r.Get("/transactions/{chain}/{hash}", h.txStatus)

	return r
}

// --- request/response types ---

type createWalletReq struct {
	Label      string `json:"label"`
	Passphrase string `json:"passphrase"`
}

type transferReq struct {
	WalletID   string `json:"wallet_id"`
	Passphrase string `json:"passphrase"`
	Chain      string `json:"chain"`
	Env        string `json:"env"`
	From       string `json:"from"`
	To         string `json:"to"`
	Amount     string `json:"amount"` // smallest unit (wei), decimal string
	Speed      string `json:"speed"`
}

// --- handlers ---

func (h *Handler) createWallet(w http.ResponseWriter, r *http.Request) {
	var req createWalletReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	wallet, err := h.wallets.Create(r.Context(), req.Label, []byte(req.Passphrase))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, wallet)
}

func (h *Handler) listWallets(w http.ResponseWriter, r *http.Request) {
	wallets, err := h.wallets.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, wallets)
}

func (h *Handler) getWallet(w http.ResponseWriter, r *http.Request) {
	wallet, err := h.wallets.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, wallet)
}

func (h *Handler) walletBalance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	chain := domain.ChainID(chi.URLParam(r, "chain"))

	amount, err := h.wallets.Balance(r.Context(), id, chain)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"chain":  string(chain),
		"amount": amount.Base.String(),
		"asset":  amount.Asset.Symbol,
	})
}

func (h *Handler) transfer(w http.ResponseWriter, r *http.Request) {
	var req transferReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(req.Amount, 10); !ok {
		http.Error(w, `{"error":"invalid amount"}`, http.StatusBadRequest)
		return
	}

	env := domain.EnvLocal
	if req.Env != "" {
		env = domain.NetworkEnv(req.Env)
	}
	speed := domain.SpeedStandard
	if req.Speed != "" {
		speed = domain.FeeSpeed(req.Speed)
	}

	hash, err := h.transactions.Transfer(r.Context(), req.WalletID, []byte(req.Passphrase), domain.TransferRequest{
		Network: domain.Network{Chain: domain.ChainID(req.Chain), Env: env},
		From:    domain.Address(req.From),
		To:      domain.Address(req.To),
		Amount:  domain.Amount{Base: amount, Asset: domain.Asset{Native: true}},
		Speed:   speed,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"hash": hash})
}

func (h *Handler) txStatus(w http.ResponseWriter, r *http.Request) {
	chain := domain.ChainID(chi.URLParam(r, "chain"))
	hash := chi.URLParam(r, "hash")

	tx, err := h.transactions.Status(r.Context(), chain, hash)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
