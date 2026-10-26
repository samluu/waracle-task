// Package email sends booking confirmation emails asynchronously: callers add a Confirmation to a Queue,
// and the queue's workers deliver it through a Sender, so a booking never waits for an email.
package email

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Confirmation is what a booking confirmation email says. Bookings don't record an email address yet, so
// there's no recipient: a real Sender would need one added to the booking.
type Confirmation struct {
	Reference  uuid.UUID
	GuestName  string
	HotelName  string
	RoomNumber string
	CheckIn    time.Time
	CheckOut   time.Time
}

// Sender delivers one confirmation. It's what a real email provider would implement.
type Sender interface {
	Send(ctx context.Context, c Confirmation) error
}

// LogSender simulates sending: it waits Delay, standing in for the time a mail provider takes, then logs
// the confirmation instead of emailing it.
type LogSender struct {
	Delay  time.Duration
	Logger *slog.Logger
}

func (s LogSender) Send(ctx context.Context, c Confirmation) error {
	// Wait with a select rather than time.Sleep so a cancelled ctx stops the wait, as a real mail
	// provider's HTTP call would. Queue.Close cancels ctx when shutdown runs out of time; with Sleep, the
	// email being sent and every queued one would still each take the full delay (and log "sent"), so
	// shutdown would wait for the whole queue and outlast its timeout.
	select {
	case <-time.After(s.Delay):
	case <-ctx.Done():
		return ctx.Err()
	}
	s.Logger.Info("confirmation email sent",
		"reference", c.Reference,
		"guest", c.GuestName,
		"hotel", c.HotelName,
		"room", c.RoomNumber,
		"check_in", c.CheckIn.Format(time.DateOnly),
		"check_out", c.CheckOut.Format(time.DateOnly),
	)
	return nil
}
