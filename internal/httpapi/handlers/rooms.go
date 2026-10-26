package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/samluu/waracle-task/internal/store"
)

const (
	defaultRoomPageSize = 20
	maxRoomPageSize     = 100
)

func (s *Server) FindRooms(ctx context.Context, req FindRoomsRequestObject) (FindRoomsResponseObject, error) {
	params := req.Params
	if params.Guests < 1 {
		return FindRooms400JSONResponse{badRequest("guests must be at least 1")}, nil
	}
	if err := validateStay(params.CheckIn.Time, params.CheckOut.Time, s.today()); err != nil {
		return FindRooms400JSONResponse{badRequest(err.Error())}, nil
	}

	limit := defaultRoomPageSize
	if params.Limit != nil {
		limit = *params.Limit
	}
	if limit < 1 || limit > maxRoomPageSize {
		return FindRooms400JSONResponse{badRequest(fmt.Sprintf("limit must be between 1 and %d", maxRoomPageSize))}, nil
	}
	var hotelSearch *string
	if params.HotelName != nil {
		search := strings.TrimSpace(*params.HotelName)
		if search == "" {
			return FindRooms400JSONResponse{badRequest("hotel_name must not be blank; omit it to search every hotel")}, nil
		}
		hotelSearch = &search
	}
	var after *store.RoomKey
	if params.Cursor != nil {
		key, err := decodeCursor(*params.Cursor)
		if err != nil {
			return FindRooms400JSONResponse{badRequest(err.Error())}, nil
		}
		after = &key
	}

	// Ask for one more room than the page holds: if it comes back, there's a next page.
	rooms, err := s.store.FindRooms(ctx, store.RoomFilter{
		HotelSearch: hotelSearch,
		CheckIn:     params.CheckIn.Time,
		CheckOut:    params.CheckOut.Time,
		Guests:      params.Guests,
		After:       after,
		Limit:       limit + 1,
	})
	if err != nil {
		return nil, err
	}

	page := FindRooms200JSONResponse{Rooms: make([]Room, 0, min(len(rooms), limit))}
	if len(rooms) > limit {
		rooms = rooms[:limit]
		last := rooms[len(rooms)-1]
		next := encodeCursor(store.RoomKey{HotelName: last.HotelName, Number: last.Number})
		page.NextCursor = &next
	}
	for _, room := range rooms {
		page.Rooms = append(page.Rooms, Room{
			Id:       room.ID,
			Number:   room.Number,
			Type:     RoomType(room.Type),
			Capacity: room.Capacity,
			Hotel:    Hotel{Id: room.HotelID, Name: room.HotelName},
		})
	}
	return page, nil
}
