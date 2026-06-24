package domain

// FeeSpeed is a chain-agnostic priority hint; each adapter maps it to its chain's fee mechanics.
type FeeSpeed string

const (
	SpeedEconomy  FeeSpeed = "economy"
	SpeedStandard FeeSpeed = "standard"
	SpeedFast     FeeSpeed = "fast"
)

type FeeEstimate struct {
	Speed FeeSpeed
	Total Amount
	// Params carries chain-specific detail (gas price+limit, sat/vB, compute units,
	// energy) and is the extension point for V2's improved estimation (ADR-0008).
	Params map[string]string
}
