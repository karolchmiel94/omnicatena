package unit_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/hdwallet"
)

// SLIP-0010 test vector 1: https://github.com/satoshilabs/slips/blob/master/slip-0010.md
var slip0010Seed, _ = hex.DecodeString("000102030405060708090a0b0c0d0e0f")

func TestDeriveKeyEd25519_SLIP0010Vectors(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"m/0'", "68e0fe46dfb67e368c75379acec591dad19df3cde26e63b93a8e704f1dade7a3"},
		{"m/0'/1'/2'/2'/1000000000'", "8f94d394a8e8fd6b1bc2f3f49f5c47e385281d5c17e65324b0f62483e37e8793"},
	}
	for _, tc := range cases {
		key, err := hdwallet.DeriveKeyEd25519(slip0010Seed, domain.DerivationPath(tc.path))
		if err != nil {
			t.Errorf("path %s: unexpected error: %v", tc.path, err)
			continue
		}
		if got := hex.EncodeToString(key); got != tc.want {
			t.Errorf("path %s:\n  got  %s\n  want %s", tc.path, got, tc.want)
		}
	}
}

func TestDeriveKeyEd25519_Deterministic(t *testing.T) {
	path := domain.DerivationPath("m/44'/501'/0'/0'")
	k1, err := hdwallet.DeriveKeyEd25519(slip0010Seed, path)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := hdwallet.DeriveKeyEd25519(slip0010Seed, path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(k1, k2) {
		t.Error("same input produced different keys")
	}
}

func TestDeriveKeyEd25519_DifferentPaths(t *testing.T) {
	k1, _ := hdwallet.DeriveKeyEd25519(slip0010Seed, "m/44'/501'/0'/0'")
	k2, _ := hdwallet.DeriveKeyEd25519(slip0010Seed, "m/44'/501'/0'/1'")
	if bytes.Equal(k1, k2) {
		t.Error("different paths should produce different keys")
	}
}

func TestDeriveKeyEd25519_UnhardenedReject(t *testing.T) {
	_, err := hdwallet.DeriveKeyEd25519(slip0010Seed, "m/0")
	if err == nil {
		t.Error("expected error for unhardened ed25519 path, got nil")
	}
}

func TestDeriveKeyEd25519_InvalidPath(t *testing.T) {
	paths := []domain.DerivationPath{
		"no-prefix",
		"m/abc'",
	}
	for _, p := range paths {
		if _, err := hdwallet.DeriveKeyEd25519(slip0010Seed, p); err == nil {
			t.Errorf("path %q: expected error, got nil", p)
		}
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	path := domain.DerivationPath("m/44'/60'/0'/0/0")
	k1, err := hdwallet.DeriveKey(slip0010Seed, path)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := hdwallet.DeriveKey(slip0010Seed, path)
	if err != nil {
		t.Fatal(err)
	}
	priv1, _ := k1.ECPrivKey()
	priv2, _ := k2.ECPrivKey()
	if !bytes.Equal(priv1.Serialize(), priv2.Serialize()) {
		t.Error("same input produced different keys")
	}
}

func TestDeriveKey_HardenedDiffersFromUnhardened(t *testing.T) {
	hardened, err := hdwallet.DeriveKey(slip0010Seed, "m/44'")
	if err != nil {
		t.Fatal(err)
	}
	unhardened, err := hdwallet.DeriveKey(slip0010Seed, "m/44")
	if err != nil {
		t.Fatal(err)
	}
	ph, _ := hardened.ECPrivKey()
	pu, _ := unhardened.ECPrivKey()
	if bytes.Equal(ph.Serialize(), pu.Serialize()) {
		t.Error("hardened and unhardened paths should produce different keys")
	}
}

func TestDeriveKey_DifferentPathsDifferentKeys(t *testing.T) {
	k1, _ := hdwallet.DeriveKey(slip0010Seed, "m/44'/60'/0'/0/0")
	k2, _ := hdwallet.DeriveKey(slip0010Seed, "m/44'/60'/0'/0/1")
	p1, _ := k1.ECPrivKey()
	p2, _ := k2.ECPrivKey()
	if bytes.Equal(p1.Serialize(), p2.Serialize()) {
		t.Error("different paths should produce different keys")
	}
}

func TestDeriveKey_InvalidPath(t *testing.T) {
	paths := []domain.DerivationPath{
		"44'/60'/0'/0/0",
		"m/abc",
		"m/",
	}
	for _, p := range paths {
		if _, err := hdwallet.DeriveKey(slip0010Seed, p); err == nil {
			t.Errorf("path %q: expected error, got nil", p)
		}
	}
}
