package httpapi

import (
	"context"
	"errors"
	"strings"

	"github.com/samluu/waracle-task/internal/store"
)

func (s *Server) FindHotels(ctx context.Context, req FindHotelsRequestObject) (FindHotelsResponseObject, error) {
	if strings.TrimSpace(req.Params.Name) == "" {
		return FindHotels400JSONResponse{badRequest("name is required")}, nil
	}
	hotel, err := s.store.FindHotelByName(ctx, req.Params.Name)
	if errors.Is(err, store.ErrNotFound) {
		return FindHotels200JSONResponse{}, nil
	}
	if err != nil {
		return nil, err
	}
	return FindHotels200JSONResponse{{Id: hotel.ID, Name: hotel.Name}}, nil
}
