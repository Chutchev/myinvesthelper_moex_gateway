package cbr

import (
	"context"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

type Service interface {
	Snapshot(ctx context.Context) (RateSnapshot, error)
}

type StubService struct{}

func NewStubService() *StubService {
	return &StubService{}
}

func (s *StubService) Snapshot(context.Context) (RateSnapshot, error) {
	return RateSnapshot{}, apperrors.ErrNotImplemented
}
