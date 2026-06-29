package moex

// BondMarketData holds market/session fields parsed from MOEX market data responses.
// Used by mergeMarketData to overlay onto Bond.
type BondMarketData struct {
	Price           *float64
	YieldToMaturity *float64
	Duration        *float64
	AccruedInterest *float64
	ValueToday      *float64
	NumTrades       *int
	MarketDataAsOf  *string
	LotSize         *int
	Currency        *string
	FaceUnit        *string
	MorningSession  *bool
	EveningSession  *bool
	WeekendSession  *bool
}

// Placeholder parser declarations — implemented in parser.go (Task 4).
// These exist here so service.go compiles against the full API surface.
// All parsers have been implemented; this file is kept for the BondMarketData struct.
