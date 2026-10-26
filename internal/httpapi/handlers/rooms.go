package handlers

import (
	"context"

	"github.com/samluu/waracle-task/internal/store"
)

func (s *Server) FindRooms(ctx context.Context, req FindRoomsRequestObject) (FindRoomsResponseObject, error) {
	params := req.Params
	if params.Guests < 1 {
		return FindRooms400JSONResponse{badRequest("guests must be at least 1")}, nil
	}
	if err := validateStay(params.CheckIn.Time, params.CheckOut.Time, s.today()); err != nil {
		return FindRooms400JSONResponse{badRequest(err.Error())}, nil
	}

	rooms, err := s.store.FindRooms(ctx, store.RoomFilter{
		HotelName: params.HotelName,
		CheckIn:   params.CheckIn.Time,
		CheckOut:  params.CheckOut.Time,
		Guests:    params.Guests,
	})
	if err != nil {
		return nil, err
	}

	resp := make(FindRooms200JSONResponse, 0, len(rooms))
	for _, room := range rooms {
		resp = append(resp, Room{
			Id:       room.ID,
			Number:   room.Number,
			Type:     RoomType(room.Type),
			Capacity: room.Capacity,
			Hotel:    Hotel{Id: room.HotelID, Name: room.HotelName},
		})
	}
	return resp, nil
}
