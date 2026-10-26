package handlers

import (
	"context"
	"errors"

	"github.com/samluu/waracle-task/internal/store"
)

// SeedData and ResetData expose the same seeding and resetting as `make seed` and `make reset`, for
// testing the API without the command line.

func (s *Server) SeedData(ctx context.Context, _ SeedDataRequestObject) (SeedDataResponseObject, error) {
	err := s.store.Seed(ctx)
	if errors.Is(err, store.ErrAlreadySeeded) {
		return SeedData409JSONResponse{Error: err.Error()}, nil
	}
	if err != nil {
		return nil, err
	}
	return SeedData204Response{}, nil
}

func (s *Server) ResetData(ctx context.Context, _ ResetDataRequestObject) (ResetDataResponseObject, error) {
	if err := s.store.Reset(ctx); err != nil {
		return nil, err
	}
	return ResetData204Response{}, nil
}
