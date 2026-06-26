package unit_test

import (
	"math/big"
	"testing"

	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/evm"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

func TestScaleTip(t *testing.T) {
	cases := []struct {
		speed domain.FeeSpeed
		input int64
		want  int64
	}{
		{domain.SpeedStandard, 1000, 1000}, // ×1.0
		{domain.SpeedFast, 1000, 1500},     // ×1.5
		{domain.SpeedEconomy, 1000, 800},   // ×0.8
		{"unknown", 1000, 1000},            // defaults to ×1.0
	}
	for _, tc := range cases {
		got := evm.ScaleTip(big.NewInt(tc.input), tc.speed)
		if got.Int64() != tc.want {
			t.Errorf("speed=%s input=%d: got %d, want %d", tc.speed, tc.input, got.Int64(), tc.want)
		}
	}
}

func TestScaleTip_ZeroTip(t *testing.T) {
	got := evm.ScaleTip(big.NewInt(0), domain.SpeedFast)
	if got.Sign() != 0 {
		t.Errorf("zero tip scaled to %s, want 0", got)
	}
}

func TestScaleTip_DoesNotMutateInput(t *testing.T) {
	base := big.NewInt(100)
	evm.ScaleTip(base, domain.SpeedFast)
	if base.Int64() != 100 {
		t.Error("ScaleTip mutated the input big.Int")
	}
}
