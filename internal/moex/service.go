package moex

import (
	"context"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

type Service interface {
	Bond(ctx context.Context, isin string) (Bond, error)
	MarketUniverse(ctx context.Context, limit int) ([]Bond, error)
}

type StubService struct{}

func NewStubService() *StubService {
	return &StubService{}
}

func (s *StubService) Bond(context.Context, string) (Bond, error) {
	return Bond{}, apperrors.ErrNotImplemented
}

func (s *StubService) MarketUniverse(context.Context, int) ([]Bond, error) {
	return nil, apperrors.ErrNotImplemented
}
