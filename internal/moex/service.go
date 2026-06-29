package moex

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cache"
	"golang.org/x/sync/errgroup"
)

const defaultCacheTTL = 15 * time.Minute

// Service defines the bond data service interface.
type Service interface {
	Bond(ctx context.Context, isin string) (Bond, error)
	MarketUniverse(ctx context.Context, limit int) (MarketUniverse, error)
}

// StubService is a placeholder implementation for bootstrapping.
type StubService struct{}

func NewStubService() *StubService {
	return &StubService{}
}

func (s *StubService) Bond(ctx context.Context, isin string) (Bond, error) {
	return Bond{}, fmt.Errorf("%w: stub service", apperrors.ErrNotImplemented)
}

func (s *StubService) MarketUniverse(ctx context.Context, limit int) (MarketUniverse, error) {
	return nil, fmt.Errorf("%w: stub service", apperrors.ErrNotImplemented)
}

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
	key := "moex:universe:" + fmt.Sprint(limit)
	var cached MarketUniverse
	if err := s.cache.Get(ctx, key, &cached); err == nil {
		return cached, nil
	} else if !errors.Is(err, apperrors.ErrCacheMiss) {
		return nil, fmt.Errorf("read universe cache: %w", err)
	}

	universePayload, err := s.client.FetchUniverse(ctx, 200)
	if err != nil {
		return nil, fmt.Errorf("fetch universe: %w", err)
	}
	snapshots, err := ParseUniverseResponse(universePayload)
	if err != nil {
		return nil, err
	}

	// Filter liquid candidates.
	var candidates []Bond
	for _, bond := range snapshots {
		if liquidBond(bond) {
			candidates = append(candidates, bond)
		}
	}
	if len(candidates) == 0 {
		result := make(MarketUniverse, 0)
		_ = s.cache.Set(ctx, key, result, s.ttl)
		return result, nil
	}

	// Sort by VALTODAY descending.
	sort.Slice(candidates, func(i, j int) bool {
		vi := 0.0
		vj := 0.0
		if candidates[i].ValueToday != nil {
			vi = *candidates[i].ValueToday
		}
		if candidates[j].ValueToday != nil {
			vj = *candidates[j].ValueToday
		}
		return vi > vj
	})

	// Truncate to limit.
	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}

	// Fan-out full bond details with bounded concurrency.
	sem := make(chan struct{}, 8)
	eg, egCtx := errgroup.WithContext(ctx)

	result := make(MarketUniverse, len(candidates))
	for i, snapshot := range candidates {
		i := i
		snapshot := snapshot
		eg.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			full, err := s.Bond(egCtx, snapshot.ISIN)
			if err != nil {
				return fmt.Errorf("bond %s: %w", snapshot.ISIN, err)
			}
			result[i] = mergeUniverseSnapshot(full, snapshot)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, key, result, s.ttl); err != nil {
		return nil, fmt.Errorf("write universe cache: %w", err)
	}
	return result, nil
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

func liquidBond(bond Bond) bool {
	return bond.FaceUnit != nil && (*bond.FaceUnit == "RUB" || *bond.FaceUnit == "SUR") &&
		bond.ValueToday != nil && *bond.ValueToday > 1_000_000 &&
		bond.NumTrades != nil && *bond.NumTrades > 10 && bond.Price != nil
}

func mergeUniverseSnapshot(full Bond, snapshot Bond) Bond {
	full.Ticker = snapshot.Ticker
	if full.Name == nil {
		full.Name = snapshot.Name
	}
	if full.ShortName == nil {
		full.ShortName = snapshot.ShortName
	}
	if full.LotSize == nil {
		full.LotSize = snapshot.LotSize
	}
	if full.FaceUnit == nil {
		full.FaceUnit = snapshot.FaceUnit
	}
	if full.Currency == nil {
		full.Currency = snapshot.Currency
	}
	if full.MorningSession == nil {
		full.MorningSession = snapshot.MorningSession
	}
	if full.EveningSession == nil {
		full.EveningSession = snapshot.EveningSession
	}
	if full.WeekendSession == nil {
		full.WeekendSession = snapshot.WeekendSession
	}
	return full
}
