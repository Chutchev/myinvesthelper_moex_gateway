package moex

import "context"

type namedMarketUniverseService interface {
	MarketUniverse(context.Context, int) (MarketUniverse, error)
}

var _ namedMarketUniverseService = NewStubService()
