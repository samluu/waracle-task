package email

import (
	"context"
	"log/slog"
	"sync"
)

// Queue holds confirmations in a buffered channel and sends them on worker goroutines.
type Queue struct {
	sender Sender
	logger *slog.Logger
	jobs   chan Confirmation

	// mu makes checking closed and sending on jobs one step. Sending on a closed channel panics, and
	// without the lock Close could close jobs between an Enqueue seeing closed == false and its send.
	// Enqueue takes the read lock, so many can run at once; Close takes the write lock, so it waits for
	// any Enqueue in progress before closing jobs. (An atomic flag wouldn't help: the check and the send
	// would still be two separate steps.)
	mu     sync.RWMutex
	closed bool

	workers sync.WaitGroup
	ctx     context.Context // passed to Send; cancelled if Close runs out of time
	cancel  context.CancelFunc
}

// NewQueue starts workers goroutines sending through sender, with room for size waiting confirmations.
func NewQueue(sender Sender, logger *slog.Logger, size, workers int) *Queue {
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{sender: sender, logger: logger, jobs: make(chan Confirmation, size), ctx: ctx, cancel: cancel}
	for range workers {
		q.workers.Go(q.work)
	}
	return q
}

func (q *Queue) work() {
	for c := range q.jobs {
		if err := q.sender.Send(q.ctx, c); err != nil {
			q.logger.Error("confirmation email not sent", "reference", c.Reference, "err", err)
		}
	}
}

// Enqueue adds c to the queue without waiting, and reports whether it was accepted. It refuses when the
// queue is full, rather than holding up the caller, or once the queue is closed.
func (q *Queue) Enqueue(c Confirmation) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.closed {
		return false
	}
	select {
	case q.jobs <- c:
		return true
	default:
		return false
	}
}

// Close stops accepting confirmations and waits for the workers to send the ones already queued. If ctx
// ends first, it cancels the sends still in progress, which then fail and are logged as not sent, and
// returns ctx's error.
func (q *Queue) Close(ctx context.Context) error {
	q.mu.Lock()
	if !q.closed {
		q.closed = true
		close(q.jobs)
	}
	q.mu.Unlock()

	done := make(chan struct{})
	go func() {
		q.workers.Wait()
		close(done)
	}()
	select {
	case <-done:
		q.cancel()
		return nil
	case <-ctx.Done():
		q.cancel()
		<-done // cancelled sends return promptly; queued ones fail fast with the cancelled context
		return ctx.Err()
	}
}
