package bitcoin

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/hdwallet"
	"github.com/karolchmiel94/omnicatena/internal/port"
)

var nativeBTC = domain.Asset{Symbol: "BTC", Decimals: 8, Native: true}

// fallback fee rates in sat/vByte when estimatesmartfee has no data (common in regtest).
var fallbackFeeRate = map[domain.FeeSpeed]int64{
	domain.SpeedEconomy:  5,
	domain.SpeedStandard: 10,
	domain.SpeedFast:     20,
}

type Config struct {
	Host        string // host:port, e.g. "localhost:18443"
	User        string
	Pass        string
	ChainParams *chaincfg.Params
}

type Adapter struct {
	cfg    Config
	client *rpcclient.Client
}

func New(cfg Config) (*Adapter, error) {
	client, err := rpcclient.New(&rpcclient.ConnConfig{
		Host:         cfg.Host,
		User:         cfg.User,
		Pass:         cfg.Pass,
		HTTPPostMode: true,
		DisableTLS:   true,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("bitcoin: connect %s: %w", cfg.Host, err)
	}
	return &Adapter{cfg: cfg, client: client}, nil
}

func (a *Adapter) Chain() domain.ChainID { return domain.ChainBitcoin }

func (a *Adapter) DeriveAccount(seed []byte, path domain.DerivationPath) (domain.Account, error) {
	key, err := hdwallet.DeriveKey(seed, path)
	if err != nil {
		return domain.Account{}, err
	}
	pubKey, err := key.ECPubKey()
	if err != nil {
		return domain.Account{}, fmt.Errorf("bitcoin: ec pub key: %w", err)
	}
	addr, err := btcutil.NewAddressPubKeyHash(
		btcutil.Hash160(pubKey.SerializeCompressed()),
		a.cfg.ChainParams,
	)
	if err != nil {
		return domain.Account{}, fmt.Errorf("bitcoin: derive address: %w", err)
	}
	return domain.Account{
		Chain:   domain.ChainBitcoin,
		Address: domain.Address(addr.EncodeAddress()),
		Path:    path,
	}, nil
}

func (a *Adapter) Balance(_ context.Context, addr domain.Address, _ domain.Asset) (domain.Amount, error) {
	utxos, err := a.fetchUTXOs(string(addr))
	if err != nil {
		return domain.Amount{}, err
	}
	total := int64(0)
	for _, u := range utxos {
		total += u.satoshis
	}
	return domain.Amount{Asset: nativeBTC, Base: big.NewInt(total)}, nil
}

func (a *Adapter) EstimateFee(_ context.Context, req domain.TransferRequest) (domain.FeeEstimate, error) {
	feeRate := fallbackFeeRate[req.Speed]

	// estimatesmartfee returns BTC/kB; convert to sat/vByte.
	raw, err := a.client.RawRequest("estimatesmartfee", []json.RawMessage{json.RawMessage(`6`)})
	if err == nil {
		var result struct {
			FeeRate float64 `json:"feerate"`
		}
		if json.Unmarshal(raw, &result) == nil && result.FeeRate > 0 {
			feeRate = int64(result.FeeRate * 1e8 / 1000) // BTC/kB → sat/vByte
		}
	}

	// P2PKH vSize: 1 input + 2 outputs (transfer + change)
	vSize := int64(1*148 + 2*34 + 10)
	totalSat := vSize * feeRate

	return domain.FeeEstimate{
		Speed: req.Speed,
		Total: domain.Amount{Asset: nativeBTC, Base: big.NewInt(totalSat)},
		Params: map[string]string{
			"fee_rate_sat_per_vbyte": fmt.Sprintf("%d", feeRate),
			"estimated_vsize":        fmt.Sprintf("%d", vSize),
		},
	}, nil
}

// unsignedPayload is what we store in domain.UnsignedTx.Payload for Bitcoin.
type unsignedPayload struct {
	RawTx        []byte   `json:"tx"`
	ScriptPubKeys [][]byte `json:"spks"` // one per input, in order
}

func (a *Adapter) BuildTransfer(ctx context.Context, req domain.TransferRequest) (domain.UnsignedTx, error) {
	fee, err := a.EstimateFee(ctx, req)
	if err != nil {
		return domain.UnsignedTx{}, err
	}
	feeSat := fee.Total.Base.Int64()

	allUTXOs, err := a.fetchUTXOs(string(req.From))
	if err != nil {
		return domain.UnsignedTx{}, err
	}
	utxos, err := a.spendable(allUTXOs)
	if err != nil {
		return domain.UnsignedTx{}, err
	}

	sendSat := req.Amount.Base.Int64()
	selected, totalIn, err := selectUTXOs(utxos, sendSat+feeSat)
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("bitcoin: %w", err)
	}

	toAddr, err := btcutil.DecodeAddress(string(req.To), a.cfg.ChainParams)
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("bitcoin: decode to address: %w", err)
	}
	fromAddr, err := btcutil.DecodeAddress(string(req.From), a.cfg.ChainParams)
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("bitcoin: decode from address: %w", err)
	}

	toScript, err := txscript.PayToAddrScript(toAddr)
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("bitcoin: to script: %w", err)
	}
	changeScript, err := txscript.PayToAddrScript(fromAddr)
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("bitcoin: change script: %w", err)
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	scriptPubKeys := make([][]byte, 0, len(selected))
	for _, u := range selected {
		hash, err := chainhash.NewHashFromStr(u.txid)
		if err != nil {
			return domain.UnsignedTx{}, fmt.Errorf("bitcoin: parse txid %s: %w", u.txid, err)
		}
		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(hash, u.vout), nil, nil))
		scriptPubKeys = append(scriptPubKeys, u.scriptPubKey)
	}

	tx.AddTxOut(wire.NewTxOut(sendSat, toScript))

	changeSat := totalIn - sendSat - feeSat
	if changeSat > 0 {
		tx.AddTxOut(wire.NewTxOut(changeSat, changeScript))
	}

	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("bitcoin: serialize tx: %w", err)
	}

	payload, err := json.Marshal(unsignedPayload{RawTx: buf.Bytes(), ScriptPubKeys: scriptPubKeys})
	if err != nil {
		return domain.UnsignedTx{}, fmt.Errorf("bitcoin: marshal payload: %w", err)
	}

	return domain.UnsignedTx{
		Chain:   domain.ChainBitcoin,
		Request: req,
		Fee:     fee,
		Payload: payload,
	}, nil
}

func (a *Adapter) Sign(_ context.Context, utx domain.UnsignedTx, signer port.Signer) (domain.SignedTx, error) {
	var p unsignedPayload
	if err := json.Unmarshal(utx.Payload, &p); err != nil {
		return domain.SignedTx{}, fmt.Errorf("bitcoin: unmarshal payload: %w", err)
	}

	var tx wire.MsgTx
	if err := tx.Deserialize(bytes.NewReader(p.RawTx)); err != nil {
		return domain.SignedTx{}, fmt.Errorf("bitcoin: deserialize tx: %w", err)
	}

	pubKey, err := signer.PublicKey(utx.Account)
	if err != nil {
		return domain.SignedTx{}, fmt.Errorf("bitcoin: get pubkey: %w", err)
	}

	for i := range tx.TxIn {
		sigHash, err := txscript.CalcSignatureHash(p.ScriptPubKeys[i], txscript.SigHashAll, &tx, i)
		if err != nil {
			return domain.SignedTx{}, fmt.Errorf("bitcoin: sig hash input %d: %w", i, err)
		}
		sig, err := signer.Sign(utx.Account, sigHash)
		if err != nil {
			return domain.SignedTx{}, fmt.Errorf("bitcoin: sign input %d: %w", i, err)
		}
		// P2PKH scriptSig: <DER sig + SIGHASH_ALL> <compressed pubkey>
		scriptSig, err := txscript.NewScriptBuilder().
			AddData(append(sig, byte(txscript.SigHashAll))).
			AddData(pubKey).
			Script()
		if err != nil {
			return domain.SignedTx{}, fmt.Errorf("bitcoin: build scriptsig input %d: %w", i, err)
		}
		tx.TxIn[i].SignatureScript = scriptSig
	}

	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return domain.SignedTx{}, fmt.Errorf("bitcoin: serialize signed tx: %w", err)
	}
	return domain.SignedTx{
		Chain: domain.ChainBitcoin,
		Raw:   buf.Bytes(),
		Hash:  tx.TxHash().String(),
	}, nil
}

func (a *Adapter) Broadcast(_ context.Context, tx domain.SignedTx) (string, error) {
	var msgTx wire.MsgTx
	if err := msgTx.Deserialize(bytes.NewReader(tx.Raw)); err != nil {
		return "", fmt.Errorf("bitcoin: deserialize: %w", err)
	}
	hash, err := a.client.SendRawTransaction(&msgTx, false)
	if err != nil {
		return "", fmt.Errorf("bitcoin: broadcast: %w", err)
	}
	return hash.String(), nil
}

func (a *Adapter) GetTransaction(_ context.Context, hash string) (domain.Transaction, error) {
	h, err := chainhash.NewHashFromStr(hash)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("bitcoin: invalid hash %q: %w", hash, err)
	}
	tx, err := a.client.GetRawTransactionVerbose(h)
	if err != nil {
		return domain.Transaction{Chain: domain.ChainBitcoin, Hash: hash, Status: domain.TxUnknown}, nil
	}
	status := domain.TxConfirmed
	if tx.Confirmations == 0 {
		status = domain.TxPending
	}
	return domain.Transaction{
		Chain:         domain.ChainBitcoin,
		Hash:          hash,
		Status:        status,
		Confirmations: tx.Confirmations,
	}, nil
}

// --- internal helpers ---

type utxo struct {
	txid         string
	vout         uint32
	satoshis     int64
	scriptPubKey []byte
	coinbase     bool
	height       int64
}

func (a *Adapter) fetchUTXOs(addr string) ([]utxo, error) {
	action, _ := json.Marshal("start")
	descriptors, _ := json.Marshal([]map[string]string{{"desc": "addr(" + addr + ")"}})

	raw, err := a.client.RawRequest("scantxoutset", []json.RawMessage{action, descriptors})
	if err != nil {
		return nil, fmt.Errorf("bitcoin: scantxoutset: %w", err)
	}

	var result struct {
		Success  bool `json:"success"`
		Unspents []struct {
			TxID         string  `json:"txid"`
			VOut         uint32  `json:"vout"`
			ScriptPubKey string  `json:"scriptPubKey"`
			Amount       float64 `json:"amount"`
			Coinbase     bool    `json:"coinbase"`
			Height       int64   `json:"height"`
		} `json:"unspents"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("bitcoin: parse scantxoutset: %w", err)
	}

	utxos := make([]utxo, 0, len(result.Unspents))
	for _, u := range result.Unspents {
		amt, err := btcutil.NewAmount(u.Amount)
		if err != nil {
			continue
		}
		spk, err := hex.DecodeString(u.ScriptPubKey)
		if err != nil {
			continue
		}
		utxos = append(utxos, utxo{
			txid:         u.TxID,
			vout:         u.VOut,
			satoshis:     int64(amt),
			scriptPubKey: spk,
			coinbase:     u.Coinbase,
			height:       u.Height,
		})
	}
	return utxos, nil
}

// spendable filters out immature coinbase UTXOs. Bitcoin requires 100 confirmations
// before coinbase outputs can be spent (COINBASE_MATURITY).
func (a *Adapter) spendable(utxos []utxo) ([]utxo, error) {
	raw, err := a.client.RawRequest("getblockcount", nil)
	if err != nil {
		return nil, fmt.Errorf("bitcoin: getblockcount: %w", err)
	}
	var tip int64
	if err := json.Unmarshal(raw, &tip); err != nil {
		return nil, fmt.Errorf("bitcoin: parse blockcount: %w", err)
	}
	out := utxos[:0]
	for _, u := range utxos {
		if u.coinbase && tip-u.height < 99 {
			continue
		}
		out = append(out, u)
	}
	return out, nil
}

// selectUTXOs picks the smallest set of UTXOs (largest-first) that covers need satoshis.
func selectUTXOs(utxos []utxo, need int64) (selected []utxo, total int64, err error) {
	sort.Slice(utxos, func(i, j int) bool { return utxos[i].satoshis > utxos[j].satoshis })
	for _, u := range utxos {
		selected = append(selected, u)
		total += u.satoshis
		if total >= need {
			return selected, total, nil
		}
	}
	return nil, 0, fmt.Errorf("insufficient funds: need %d sat, have %d sat", need, total)
}
