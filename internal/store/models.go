package store

import "time"

type RoomType string

const (
	RoomTypeSingle RoomType = "single"
	RoomTypeDouble RoomType = "double"
	RoomTypeDeluxe RoomType = "deluxe"
)

type Hotel struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

type Room struct {
	ID      int64    `db:"id"`
	HotelID int64    `db:"hotel_id"`
	Number  string   `db:"number"`
	Type    RoomType `db:"type"`
	// Capacity comes from room_types; queries join it in.
	Capacity int `db:"capacity"`
}

type Booking struct {
	Reference string `db:"reference"`
	RoomID    int64  `db:"room_id"`
	GuestName string `db:"guest_name"`
	Guests    int    `db:"guests"`
	// CheckIn and CheckOut are dates (midnight UTC); the stay covers the nights [CheckIn, CheckOut).
	CheckIn   time.Time `db:"check_in"`
	CheckOut  time.Time `db:"check_out"`
	CreatedAt time.Time `db:"created_at"`
}
