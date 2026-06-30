//go:build smoke

package smoke_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/tron"
	"github.com/karolchmiel94/omnicatena/internal/adapter/keystore"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

const tronRPCURL = "http://localhost:9090"

// getTronAdminPrivKey parses the plain-text /admin/accounts page from tron-quickstart
// and returns the private key for account index 0.
func getTronAdminPrivKey(t *testing.T) string {
	t.Helper()
	resp, err := http.Get(tronRPCURL + "/admin/accounts")
	if err != nil {
		t.Fatalf("GET /admin/accounts: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	inPrivKeys := false
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Private Keys") {
			inPrivKeys = true
			continue
		}
		if inPrivKeys && strings.HasPrefix(line, "(0) ") {
			return strings.TrimPrefix(line, "(0) ")
		}
	}
	t.Fatal("could not parse private key 0 from /admin/accounts")
	return ""
}

// privKeyToTronAddr derives the T... address from a raw hex private key.
func privKeyToTronAddr(privKeyHex string) domain.Address {
	privKey, _ := crypto.HexToECDSA(privKeyHex)
	ethAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	return domain.Address(base58.CheckEncode(ethAddr.Bytes(), 0x41))
}

func tronAddrToHex(t *testing.T, addr domain.Address) string {
	t.Helper()
	raw, ver, err := base58.CheckDecode(string(addr))
	if err != nil {
		t.Fatalf("decode tron address %s: %v", addr, err)
	}
	return fmt.Sprintf("%02x%s", ver, hex.EncodeToString(raw))
}

// fundTRON transfers amount sun to toAddr, signing with a raw private key.
// Used for test setup only — not going through our keystore.
func fundTRON(ctx context.Context, t *testing.T, from, to domain.Address, privKeyHex string, amount int64) {
	t.Helper()
	fromHex := tronAddrToHex(t, from)
	toHex := tronAddrToHex(t, to)

	body, _ := json.Marshal(map[string]any{
		"owner_address": fromHex,
		"to_address":    toHex,
		"amount":        amount,
		"visible":       false,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, tronRPCURL+"/wallet/createtransaction", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("fund createtransaction: %v", err)
	}
	defer resp.Body.Close()

	var rawTx map[string]json.RawMessage
	json.NewDecoder(resp.Body).Decode(&rawTx)

	var txID string
	json.Unmarshal(rawTx["txID"], &txID)
	digest, _ := hex.DecodeString(txID)

	privKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		t.Fatalf("parse fund privkey: %v", err)
	}
	sig, _ := crypto.Sign(digest, privKey)
	sigJSON, _ := json.Marshal([]string{hex.EncodeToString(sig)})
	rawTx["signature"] = sigJSON

	signed, _ := json.Marshal(rawTx)
	req2, _ := http.NewRequestWithContext(ctx, http.MethodPost, tronRPCURL+"/wallet/broadcasttransaction", bytes.NewReader(signed))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("fund broadcast: %v", err)
	}
	defer resp2.Body.Close()
	var result struct {
		Result bool `json:"result"`
	}
	json.NewDecoder(resp2.Body).Decode(&result)
	if !result.Result {
		t.Fatal("fund broadcast failed — check that the admin account has sufficient balance")
	}
}

func TestSmokeEndToEnd(t *testing.T) {
	ctx := context.Background()
	adapter := tron.New(tron.Config{RPCURL: tronRPCURL})

	ks := keystore.New()
	seed, err := ks.Create("smoke", []byte("test"))
	if err != nil {
		t.Fatalf("ks.Create: %v", err)
	}

	acc, err := adapter.DeriveAccount(seed, "m/44'/195'/0'/0/0")
	if err != nil {
		t.Fatalf("DeriveAccount: %v", err)
	}
	t.Logf("address: %s", acc.Address)

	// Fund from quickstart admin account 0.
	adminPrivKey := getTronAdminPrivKey(t)
	adminAddr := privKeyToTronAddr(adminPrivKey)
	t.Logf("admin: %s", adminAddr)

	const oneTRX = int64(1_000_000)
	fundTRON(ctx, t, adminAddr, acc.Address, adminPrivKey, 10*oneTRX)
	time.Sleep(3 * time.Second)

	bal, err := adapter.Balance(ctx, acc.Address, domain.Asset{})
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	t.Logf("balance: %s sun", bal.Base)
	if bal.Base.Cmp(big.NewInt(0)) == 0 {
		t.Fatal("expected non-zero balance after funding")
	}

	// Transfer 1 TRX to a fresh account at index 1.
	recip, err := adapter.DeriveAccount(seed, "m/44'/195'/0'/0/1")
	if err != nil {
		t.Fatalf("DeriveAccount recipient: %v", err)
	}
	t.Logf("recipient: %s", recip.Address)

	req := domain.TransferRequest{
		Network: domain.Network{Chain: domain.ChainTron},
		From:    acc.Address,
		To:      recip.Address,
		Amount:  domain.Amount{Asset: domain.Asset{Symbol: "TRX", Decimals: 6, Native: true}, Base: big.NewInt(oneTRX)},
		Speed:   domain.SpeedStandard,
	}

	unsigned, err := adapter.BuildTransfer(ctx, req)
	if err != nil {
		t.Fatalf("BuildTransfer: %v", err)
	}
	unsigned.Account = acc

	signer, err := ks.Signer("smoke", []byte("test"))
	if err != nil {
		t.Fatalf("Signer: %v", err)
	}

	signed, err := adapter.Sign(ctx, unsigned, signer)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	t.Logf("signed txID: %s", signed.Hash)

	txHash, err := adapter.Broadcast(ctx, signed)
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	t.Logf("broadcast: %s", txHash)
	time.Sleep(4 * time.Second)

	txInfo, err := adapter.GetTransaction(ctx, txHash)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	t.Logf("status=%s block=%d", txInfo.Status, txInfo.BlockHeight)
	if txInfo.Status != domain.TxConfirmed {
		t.Errorf("expected TxConfirmed, got %s", txInfo.Status)
	}

	recipBal, err := adapter.Balance(ctx, recip.Address, domain.Asset{})
	if err != nil {
		t.Fatalf("recipient balance: %v", err)
	}
	t.Logf("recipient balance: %s sun", recipBal.Base)
	if recipBal.Base.Cmp(big.NewInt(oneTRX)) != 0 {
		t.Errorf("expected %d sun, got %s", oneTRX, recipBal.Base)
	}
}
