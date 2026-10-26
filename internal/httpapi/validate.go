package httpapi

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samluu/waracle-task/internal/store"
)

const (
	// maxAdvanceYears is how far ahead a stay can start.
	maxAdvanceYears = 1
	// maxStayNights is the longest stay that can be booked.
	maxStayNights = 30
)

// validateStay checks a stay's dates against today (a UTC date at midnight). It's shared by
// search and booking so a search never offers a stay that booking would then reject.
func validateStay(checkIn, checkOut, today time.Time) error {
	switch {
	case checkIn.IsZero():
		return errors.New("check_in is required")
	case checkOut.IsZero():
		return errors.New("check_out is required")
	case !checkOut.After(checkIn):
		return errors.New("check_out must be after check_in")
	case checkOut.After(checkIn.AddDate(0, 0, maxStayNights)):
		return fmt.Errorf("a stay can be at most %d nights", maxStayNights)
	case checkIn.Before(today):
		return errors.New("check_in must not be in the past")
	case checkIn.After(today.AddDate(maxAdvanceYears, 0, 0)):
		return errors.New("check_in must be no more than a year from today")
	}
	return nil
}

// toNewBooking validates a booking request and turns it into what the store books, with the guest
// name trimmed. The handler passes its result straight to the store, so no rule here can be skipped.
func toNewBooking(req BookingRequest, today time.Time) (store.NewBooking, error) {
	if err := validateBookingRequest(req); err != nil {
		return store.NewBooking{}, err
	}
	if err := validateStay(req.CheckIn.Time, req.CheckOut.Time, today); err != nil {
		return store.NewBooking{}, err
	}
	return store.NewBooking{
		RoomID:    req.RoomId,
		GuestName: strings.TrimSpace(req.GuestName),
		Guests:    req.Guests,
		CheckIn:   req.CheckIn.Time,
		CheckOut:  req.CheckOut.Time,
	}, nil
}

func validateBookingRequest(req BookingRequest) error {
	switch {
	case req.RoomId <= 0:
		return errors.New("room_id is required")
	case strings.TrimSpace(req.GuestName) == "":
		return errors.New("guest_name is required")
	case req.Guests < 1:
		return errors.New("guests must be at least 1")
	}
	return nil
}

// today returns the current UTC date at midnight, matching how dates are parsed from requests.
func (s *Server) today() time.Time {
	y, m, d := s.now().UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
