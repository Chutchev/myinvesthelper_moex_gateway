package moex

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cache"
)

const defaultCacheTTL = 15 * time.Minute

type CachedService struct {
	client Client
	cache  cache.Cache
	ttl    time.Duration
}

func NewService(client Client, c cache.Cache, ttl time.Duration) *CachedService {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	return &CachedService{client: client, cache: c, ttl: ttl}
}

func (s *CachedService) Bond(ctx context.Context, isin string) (Bond, error) {
	key := "moex:bond:" + isin
	var cached Bond
	if err := s.cache.Get(ctx, key, &cached); err == nil {
		return cached, nil
	} else if !errors.Is(err, apperrors.ErrCacheMiss) {
		return Bond{}, fmt.Errorf("read bond cache: %w", err)
	}

	descriptionPayload, err := s.client.FetchDescription(ctx, isin)
	if err != nil {
		return Bond{}, fmt.Errorf("fetch description: %w", err)
	}
	bond, err := ParseBondDescription(descriptionPayload)
	if err != nil {
		return Bond{}, err
	}
	bond.ISIN = isin

	marketPayload, err := s.client.FetchMarketData(ctx, isin)
	if err != nil {
		return Bond{}, fmt.Errorf("fetch market data: %w", err)
	}
	market, err := ParseMarketData(marketPayload)
	if err != nil {
		return Bond{}, err
	}
	mergeMarketData(&bond, market)

	bondizationPayload, err := s.client.FetchBondization(ctx, isin)
	if err != nil {
		return Bond{}, fmt.Errorf("fetch bondization: %w", err)
	}
	bond.CouponCalendar, bond.CashflowSchedule, err = ParseBondization(bondizationPayload)
	if err != nil {
		return Bond{}, err
	}

	if err := s.cache.Set(ctx, key, bond, s.ttl); err != nil {
		return Bond{}, fmt.Errorf("write bond cache: %w", err)
	}
	return bond, nil
}

func (s *CachedService) MarketUniverse(ctx context.Context, limit int) (MarketUniverse, error) {
	return nil, apperrors.ErrNotImplemented
}

func mergeMarketData(bond *Bond, market *BondMarketData) {
	if market == nil {
		return
	}
	if market.Price != nil {
		bond.Price = market.Price
	}
	if market.YieldToMaturity != nil {
		bond.YieldToMaturity = market.YieldToMaturity
	}
	if market.Duration != nil {
		bond.Duration = market.Duration
	}
	if market.AccruedInterest != nil {
		bond.AccruedInterest = market.AccruedInterest
	}
	if market.ValueToday != nil {
		bond.ValueToday = market.ValueToday
	}
	if market.NumTrades != nil {
		bond.NumTrades = market.NumTrades
	}
	if market.MarketDataAsOf != nil {
		bond.MarketDataAsOf = market.MarketDataAsOf
	}
	if market.LotSize != nil {
		bond.LotSize = market.LotSize
	}
	if market.Currency != nil {
		bond.Currency = market.Currency
	}
	if market.FaceUnit != nil && bond.FaceUnit == nil {
		bond.FaceUnit = market.FaceUnit
	}
	if market.MorningSession != nil {
		bond.MorningSession = market.MorningSession
	}
	if market.EveningSession != nil {
		bond.EveningSession = market.EveningSession
	}
	if market.WeekendSession != nil {
		bond.WeekendSession = market.WeekendSession
	}
}
