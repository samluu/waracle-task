package handlers

import (
	"context"
	"strings"

	"github.com/samluu/waracle-task/internal/store"
)

func (s *Server) FindHotels(ctx context.Context, req FindHotelsRequestObject) (FindHotelsResponseObject, error) {
	params := req.Params
	search := strings.TrimSpace(params.Name)
	if search == "" {
		return FindHotels400JSONResponse{badRequest("name is required")}, nil
	}
	limit, err := pageSize(params.Limit)
	if err != nil {
		return FindHotels400JSONResponse{badRequest(err.Error())}, nil
	}
	var after *string
	if params.Cursor != nil {
		c, err := decodeCursor[hotelCursor](*params.Cursor)
		if err != nil {
			return FindHotels400JSONResponse{badRequest(err.Error())}, nil
		}
		after = &c.HotelName
	}

	// Ask for one more hotel than the page holds: if it comes back, there's a next page.
	hotels, err := s.store.FindHotels(ctx, store.HotelFilter{Search: search, After: after, Limit: limit + 1})
	if err != nil {
		return nil, err
	}

	page := FindHotels200JSONResponse{Hotels: make([]Hotel, 0, min(len(hotels), limit))}
	if len(hotels) > limit {
		hotels = hotels[:limit]
		next := encodeCursor(hotelCursor{HotelName: hotels[len(hotels)-1].Name})
		page.NextCursor = &next
	}
	for _, h := range hotels {
		page.Hotels = append(page.Hotels, Hotel{Id: h.ID, Name: h.Name})
	}
	return page, nil
}
