package tron

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/hdwallet"
	"github.com/karolchmiel94/omnicatena/internal/port"
)

var nativeTRX = domain.Asset{Symbol: "TRX", Decimals: 6, Native: true}

type Config struct {
	RPCURL string
}

type Adapter struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Adapter {
	return &Adapter{cfg: cfg, client: &http.Client{}}
}

func (a *Adapter) Chain() domain.ChainID { return domain.ChainTron }

func (a *Adapter) DeriveAccount(seed []byte, path domain.DerivationPath) (domain.Account, error) {
	key, err := hdwallet.DeriveKey(seed, path)
	if err != nil {
		return domain.Account{}, err
	}
	privKey, err := key.ECPrivKey()
	if err != nil {
		return domain.Account{}, fmt.Errorf("tron: ec priv key: %w", err)
	}
	// TRON uses the same keccak20 hash as Ethereum, encoded as base58check with 0x41 prefix.
	ethAddr := crypto.PubkeyToAddress(privKey.ToECDSA().PublicKey)
	addr := base58.CheckEncode(ethAddr.Bytes(), 0x41)
	return domain.Account{
		Chain:   domain.ChainTron,
		Address: domain.Address(addr),
		Path:    path,
	}, nil
}

func (a *Adapter) Balance(ctx context.Context, addr domain.Address, _ domain.Asset) (domain.Amount, error) {
	hexA, err := toHexAddr(addr)
	if err != nil {
		return domain.Amount{}, err
	}
	var resp struct {
		Balance int64 `json:"balance"`
	}
	if err := a.post(ctx, "/wallet/getaccount", map[string]any{"address": hexA, "visible": false}, &resp); err != nil {
		return domain.Amount{}, fmt.Errorf("tron: balance: %w", err)
	}
	return domain.Amount{Asset: nativeTRX, Base: big.NewInt(resp.Balance)}, nil
}

// EstimateFee returns the worst-case bandwidth cost for a native TRX transfer.
// 268 bytes × 1000 sun/byte = 268,000 sun. Actual cost is 0 if the account has
// free daily bandwidth (1500 points/day), but we return the ceiling for safety.
func (a *Adapter) EstimateFee(_ context.Context, req domain.TransferRequest) (domain.FeeEstimate, error) {
	return domain.FeeEstimate{
		Speed: req.Speed,
		Total: domain.Amount{Asset: nativeTRX, Base: big.NewInt(268_000)},
		Params: map[string]string{
			"bandwidth_bytes": "268",
			"bandwidth_price": "1000",
		},
	}, nil
}

func (a *Adapter) BuildTransfer(ctx context.Context, req domain.TransferRequest) (domain.UnsignedTx, error) {
	fromHex, err := toHexAddr(req.From)
	if err != nil {
		return domain.UnsignedTx{}, err
	}
	toHex, err := toHexAddr(req.To)
	if err != nil {
		return domain.UnsignedTx{}, err
	}

	var rawTx json.RawMessage
	if err := a.post(ctx, "/wallet/createtransaction", map[string]any{
		"owner_address": fromHex,
		"to_address":    toHex,
		"amount":        req.Amount.Base.Int64(),
		"visible":       false,
	}, &rawTx); err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("tron: create tx: %w", err)
	}
	// TRON returns HTTP 200 with {"Error": "..."} for contract validation failures.
	var apiErr struct{ Error string }
	if json.Unmarshal(rawTx, &apiErr) == nil && apiErr.Error != "" {
		return domain.UnsignedTx{}, fmt.Errorf("tron: create tx: %s", apiErr.Error)
	}

	fee, _ := a.EstimateFee(ctx, req)
	return domain.UnsignedTx{
		Chain:   domain.ChainTron,
		Request: req,
		Fee:     fee,
		Payload: rawTx,
	}, nil
}

func (a *Adapter) Sign(_ context.Context, utx domain.UnsignedTx, signer port.Signer) (domain.SignedTx, error) {
	var tx map[string]json.RawMessage
	if err := json.Unmarshal(utx.Payload, &tx); err != nil {
		return domain.SignedTx{}, fmt.Errorf("tron: parse tx: %w", err)
	}

	var txID string
	if err := json.Unmarshal(tx["txID"], &txID); err != nil {
		return domain.SignedTx{}, fmt.Errorf("tron: parse txID: %w", err)
	}
	digest, err := hex.DecodeString(txID)
	if err != nil {
		return domain.SignedTx{}, fmt.Errorf("tron: decode txID: %w", err)
	}

	sig, err := signer.Sign(utx.Account, digest)
	if err != nil {
		return domain.SignedTx{}, fmt.Errorf("tron: sign: %w", err)
	}

	sigJSON, _ := json.Marshal([]string{hex.EncodeToString(sig)})
	tx["signature"] = sigJSON

	raw, err := json.Marshal(tx)
	if err != nil {
		return domain.SignedTx{}, fmt.Errorf("tron: marshal signed tx: %w", err)
	}
	return domain.SignedTx{Chain: domain.ChainTron, Raw: raw, Hash: txID}, nil
}

func (a *Adapter) Broadcast(ctx context.Context, tx domain.SignedTx) (string, error) {
	var resp struct {
		Result  bool   `json:"result"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := a.postRaw(ctx, "/wallet/broadcasttransaction", tx.Raw, &resp); err != nil {
		return "", fmt.Errorf("tron: broadcast: %w", err)
	}
	if !resp.Result {
		return "", fmt.Errorf("tron: broadcast failed: %s %s", resp.Code, resp.Message)
	}
	return tx.Hash, nil
}

func (a *Adapter) GetTransaction(ctx context.Context, hash string) (domain.Transaction, error) {
	var txResp map[string]json.RawMessage
	if err := a.post(ctx, "/wallet/gettransactionbyid", map[string]any{"value": hash}, &txResp); err != nil {
		return domain.Transaction{}, fmt.Errorf("tron: get tx: %w", err)
	}
	if len(txResp) == 0 || txResp["txID"] == nil {
		return domain.Transaction{Chain: domain.ChainTron, Hash: hash, Status: domain.TxUnknown}, nil
	}

	var info struct {
		BlockNumber int64 `json:"blockNumber"`
		Receipt     struct {
			Result string `json:"result"`
		} `json:"receipt"`
	}
	_ = a.post(ctx, "/wallet/gettransactioninfobyid", map[string]any{"value": hash}, &info)

	status := domain.TxPending
	if info.BlockNumber > 0 {
		status = domain.TxConfirmed
		if info.Receipt.Result == "FAILED" {
			status = domain.TxFailed
		}
	}
	return domain.Transaction{
		Chain:       domain.ChainTron,
		Hash:        hash,
		Status:      status,
		BlockHeight: uint64(info.BlockNumber),
	}, nil
}

// toHexAddr converts a base58check TRON address (T...) to the 41-prefixed hex used by the API.
func toHexAddr(addr domain.Address) (string, error) {
	raw, ver, err := base58.CheckDecode(string(addr))
	if err != nil {
		return "", fmt.Errorf("tron: bad address %s: %w", addr, err)
	}
	return fmt.Sprintf("%02x%s", ver, hex.EncodeToString(raw)), nil
}

func (a *Adapter) post(ctx context.Context, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return a.postRaw(ctx, path, b, out)
}

func (a *Adapter) postRaw(ctx context.Context, path string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.RPCURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
