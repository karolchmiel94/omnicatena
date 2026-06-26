package unit_test

import (
	"bytes"
	"crypto/ed25519"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/karolchmiel94/omnicatena/internal/adapter/keystore"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

func TestKeystore_CreateUnlockRoundtrip(t *testing.T) {
	ks := keystore.New()
	pass := []byte("testpass")

	seed, err := ks.Create("w1", pass)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(seed) == 0 {
		t.Fatal("Create returned empty seed")
	}

	got, err := ks.Unlock("w1", pass)
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if !bytes.Equal(seed, got) {
		t.Error("Unlock returned different seed than Create")
	}
}

func TestKeystore_WrongPassphrase(t *testing.T) {
	ks := keystore.New()
	if _, err := ks.Create("w", []byte("correct")); err != nil {
		t.Fatal(err)
	}
	_, err := ks.Unlock("w", []byte("wrong"))
	if err == nil {
		t.Error("expected error with wrong passphrase")
	}
}

func TestKeystore_NotFound(t *testing.T) {
	ks := keystore.New()
	_, err := ks.Unlock("missing", []byte("pass"))
	if err == nil {
		t.Error("expected error for unknown wallet ID")
	}
}

func TestKeystore_UniqueSeeds(t *testing.T) {
	ks := keystore.New()
	pass := []byte("pass")
	s1, _ := ks.Create("w1", pass)
	s2, _ := ks.Create("w2", pass)
	if bytes.Equal(s1, s2) {
		t.Error("two Create calls produced the same seed")
	}
}

func TestSigner_Solana(t *testing.T) {
	ks := keystore.New()
	if _, err := ks.Create("w", []byte("pass")); err != nil {
		t.Fatal(err)
	}
	signer, err := ks.Signer("w", []byte("pass"))
	if err != nil {
		t.Fatal(err)
	}

	acc := domain.Account{
		Chain: domain.ChainSolana,
		Path:  "m/44'/501'/0'/0'",
	}
	msg := []byte("hello omnicatena")

	sig, err := signer.Sign(acc, msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if len(sig) != ed25519.SignatureSize {
		t.Errorf("signature length: got %d, want %d", len(sig), ed25519.SignatureSize)
	}

	pub, err := signer.PublicKey(acc)
	if err != nil {
		t.Fatalf("PublicKey: %v", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Errorf("pubkey length: got %d, want %d", len(pub), ed25519.PublicKeySize)
	}

	if !ed25519.Verify(ed25519.PublicKey(pub), msg, sig) {
		t.Error("ed25519 signature verification failed")
	}
}

func TestSigner_EVM(t *testing.T) {
	ks := keystore.New()
	if _, err := ks.Create("w", []byte("pass")); err != nil {
		t.Fatal(err)
	}
	signer, err := ks.Signer("w", []byte("pass"))
	if err != nil {
		t.Fatal(err)
	}

	acc := domain.Account{
		Chain: domain.ChainEthereum,
		Path:  "m/44'/60'/0'/0/0",
	}
	digest := crypto.Keccak256([]byte("hello omnicatena"))

	sig, err := signer.Sign(acc, digest)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if len(sig) != 65 {
		t.Errorf("signature length: got %d, want 65", len(sig))
	}

	pub, err := signer.PublicKey(acc)
	if err != nil {
		t.Fatalf("PublicKey: %v", err)
	}
	if len(pub) != 33 {
		t.Errorf("pubkey length: got %d, want 33 (compressed secp256k1)", len(pub))
	}

	// Recover the signer from the signature and compare to PublicKey().
	recovered, err := crypto.SigToPub(digest, sig)
	if err != nil {
		t.Fatalf("SigToPub: %v", err)
	}
	if !bytes.Equal(crypto.CompressPubkey(recovered), pub) {
		t.Error("recovered EVM public key does not match PublicKey()")
	}
}

func TestSigner_Bitcoin(t *testing.T) {
	ks := keystore.New()
	if _, err := ks.Create("w", []byte("pass")); err != nil {
		t.Fatal(err)
	}
	signer, err := ks.Signer("w", []byte("pass"))
	if err != nil {
		t.Fatal(err)
	}

	acc := domain.Account{
		Chain: domain.ChainBitcoin,
		Path:  "m/44'/0'/0'/0/0",
	}
	// Use a 32-byte hash as the sighash preimage representative.
	digest := crypto.Keccak256([]byte("sighash preimage"))

	sig, err := signer.Sign(acc, digest)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	// DER-encoded secp256k1 signature: 70–72 bytes.
	if len(sig) < 70 || len(sig) > 72 {
		t.Errorf("DER signature length: got %d, want 70–72", len(sig))
	}

	pub, err := signer.PublicKey(acc)
	if err != nil {
		t.Fatalf("PublicKey: %v", err)
	}
	if len(pub) != 33 {
		t.Errorf("pubkey length: got %d, want 33 (compressed secp256k1)", len(pub))
	}
}

func TestSigner_Deterministic(t *testing.T) {
	ks := keystore.New()
	if _, err := ks.Create("w", []byte("pass")); err != nil {
		t.Fatal(err)
	}

	acc := domain.Account{Chain: domain.ChainEthereum, Path: "m/44'/60'/0'/0/0"}

	s1, _ := ks.Signer("w", []byte("pass"))
	s2, _ := ks.Signer("w", []byte("pass"))

	pub1, _ := s1.PublicKey(acc)
	pub2, _ := s2.PublicKey(acc)
	if !bytes.Equal(pub1, pub2) {
		t.Error("same wallet/passphrase produced different public keys")
	}
}
