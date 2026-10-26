package handlers

import (
	"context"
	"strings"
)

func (s *Server) FindHotels(ctx context.Context, req FindHotelsRequestObject) (FindHotelsResponseObject, error) {
	search := strings.TrimSpace(req.Params.Name)
	if search == "" {
		return FindHotels400JSONResponse{badRequest("name is required")}, nil
	}
	hotels, err := s.store.FindHotels(ctx, search)
	if err != nil {
		return nil, err
	}
	resp := make(FindHotels200JSONResponse, 0, len(hotels))
	for _, h := range hotels {
		resp = append(resp, Hotel{Id: h.ID, Name: h.Name})
	}
	return resp, nil
}
