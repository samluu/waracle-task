package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func confirmation(n int) Confirmation {
	return Confirmation{
		Reference: uuid.MustParse(fmt.Sprintf("5eed0000-0000-4000-8000-%012d", n)),
		GuestName: "Jane Doe", HotelName: "The Grand Hotel", RoomNumber: "102",
		CheckIn: time.Date(2027, 1, 11, 0, 0, 0, 0, time.UTC), CheckOut: time.Date(2027, 1, 13, 0, 0, 0, 0, time.UTC),
	}
}

// logs returns a logger writing JSON lines to a buffer, and a function that parses what it wrote.
func logs(t *testing.T) (*slog.Logger, func() []map[string]any) {
	var mu sync.Mutex
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(lockedWriter{&mu, &buf}, nil))
	return logger, func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		var lines []map[string]any
		for line := range strings.Lines(buf.String()) {
			var m map[string]any
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				t.Fatalf("log line %q: %v", line, err)
			}
			lines = append(lines, m)
		}
		return lines
	}
}

type lockedWriter struct {
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func (w lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func TestLogSenderLogsAfterDelay(t *testing.T) {
	logger, logged := logs(t)
	start := time.Now()
	if err := (LogSender{Delay: 50 * time.Millisecond, Logger: logger}).Send(t.Context(), confirmation(1)); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Fatalf("sent after %v, want at least the 50ms delay", elapsed)
	}
	lines := logged()
	if len(lines) != 1 {
		t.Fatalf("want 1 log line, got %v", lines)
	}
	want := map[string]any{
		"msg": "confirmation email sent", "reference": "5eed0000-0000-4000-8000-000000000001", "guest": "Jane Doe",
		"hotel": "The Grand Hotel", "room": "102", "check_in": "2027-01-11", "check_out": "2027-01-13",
	}
	for k, v := range want {
		if lines[0][k] != v {
			t.Errorf("log %s = %v, want %v", k, lines[0][k], v)
		}
	}
}

func TestLogSenderStopsWhenCancelled(t *testing.T) {
	logger, logged := logs(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	err := (LogSender{Delay: time.Hour, Logger: logger}).Send(ctx, confirmation(1))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
	if lines := logged(); len(lines) != 0 {
		t.Fatalf("a cancelled send was logged as sent: %v", lines)
	}
}

// recorder is a Sender that records what it sends. If block is set, each Send signals started and
// then waits for block to close, so tests can hold workers busy.
type recorder struct {
	mu      sync.Mutex
	sent    []uuid.UUID
	started chan struct{}
	block   chan struct{}
}

func (r *recorder) Send(_ context.Context, c Confirmation) error {
	if r.block != nil {
		r.started <- struct{}{}
		<-r.block
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, c.Reference)
	return nil
}

func TestQueueSendsEverythingQueued(t *testing.T) {
	r := &recorder{}
	logger, _ := logs(t)
	q := NewQueue(r, logger, 10, 3)
	var want []uuid.UUID
	for i := 1; i <= 5; i++ {
		if !q.Enqueue(confirmation(i)) {
			t.Fatalf("confirmation %d not accepted", i)
		}
		want = append(want, confirmation(i).Reference)
	}
	if err := q.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(r.sent, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	if !slices.Equal(r.sent, want) {
		t.Fatalf("sent %v, want %v", r.sent, want)
	}
}

// Enqueue never waits: with the one worker busy and the one buffer slot taken, the next is refused.
func TestQueueRefusesWhenFull(t *testing.T) {
	r := &recorder{started: make(chan struct{}), block: make(chan struct{})}
	logger, _ := logs(t)
	q := NewQueue(r, logger, 1, 1)

	if !q.Enqueue(confirmation(1)) {
		t.Fatal("first not accepted")
	}
	<-r.started // the worker has taken the first and is busy with it
	if !q.Enqueue(confirmation(2)) {
		t.Fatal("second not accepted into the buffer")
	}
	if q.Enqueue(confirmation(3)) {
		t.Fatal("third accepted into a full queue")
	}

	go func() { <-r.started }() // let the second start once the first finishes
	close(r.block)
	if err := q.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(r.sent) != 2 {
		t.Fatalf("sent %d, want the 2 accepted", len(r.sent))
	}
}

func TestQueueRefusesAfterClose(t *testing.T) {
	logger, _ := logs(t)
	q := NewQueue(&recorder{}, logger, 10, 1)
	if err := q.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if q.Enqueue(confirmation(1)) {
		t.Fatal("accepted after Close")
	}
	if err := q.Close(t.Context()); err != nil { // closing twice is harmless
		t.Fatal(err)
	}
}

// If Close runs out of time, sends in progress are cancelled and logged as not sent, rather than
// keeping shutdown waiting.
func TestQueueCloseGivesUpAtDeadline(t *testing.T) {
	logger, logged := logs(t)
	q := NewQueue(LogSender{Delay: time.Hour, Logger: logger}, logger, 10, 1)
	q.Enqueue(confirmation(1))
	q.Enqueue(confirmation(2))

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if err := q.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
	var notSent int
	for _, line := range logged() {
		if line["msg"] == "confirmation email not sent" {
			notSent++
		}
		if line["msg"] == "confirmation email sent" {
			t.Errorf("logged as sent: %v", line)
		}
	}
	if notSent != 2 {
		t.Fatalf("want both logged as not sent, got %d: %v", notSent, logged())
	}
}
