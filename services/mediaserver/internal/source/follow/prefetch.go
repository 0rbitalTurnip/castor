package follow

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"time"

	"github.com/stupside/castor/services/mediaserver/internal/source/timeline"
)

const (
	// parallel is how many segments are read from the origin at once, since an origin throttles each connection.
	parallel = 4

	// ahead is how much media past the reader's last request is read before it is asked for.
	ahead = 30 * time.Second

	// attempts is how many requests one segment may have in flight, the first and its hedges.
	attempts = 3

	// hedgeFloor and hedgeDefault bound when a slow read is asked for again; recent reads set it in between.
	hedgeFloor   = 3 * time.Second
	hedgeDefault = 8 * time.Second

	// remembered is how many recent reads the hedge deadline is taken from.
	remembered = 8
)

// slot is one segment read ahead of the reader: its bytes once done closes.
type slot struct {
	done chan struct{}
	body []byte
	err  error
}

// fetched is a segment's bytes, decrypted, read ahead when the reader asked for one before it; it reads ahead past it.
func (f *feed) fetched(ctx context.Context, sequence int64, s timeline.Segment) ([]byte, error) {
	f.mu.Lock()
	f.reached = max(f.reached, sequence)
	sl, ok := f.slots[sequence]
	if !ok {
		// The reader is waiting on it, so it does not wait for a free slot.
		sl = f.launch(sequence, s)
	}
	f.pump()
	f.mu.Unlock()

	select {
	case <-sl.done:
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for held := range f.slots {
		// One behind is kept for a reader that retries the segment it just failed to finish.
		if held < sequence-1 {
			delete(f.slots, held)
		}
	}
	if sl.err != nil && f.slots[sequence] == sl {
		delete(f.slots, sequence)
	}
	return sl.body, sl.err
}

// pump starts reads past the reader, playhead order first, until the window is covered or every slot is busy; f.mu is held.
func (f *feed) pump() {
	var covered time.Duration
	for next := f.reached + 1; covered < ahead && f.inFlight < parallel; next++ {
		s, listed := f.ledger.Entry(next)
		if !listed {
			return
		}
		covered += s.Duration
		if _, held := f.slots[next]; !held {
			f.launch(next, s)
		}
	}
}

// launch starts reading one segment under the feed's lifetime; f.mu is held.
func (f *feed) launch(sequence int64, s timeline.Segment) *slot {
	sl := &slot{done: make(chan struct{})}
	f.slots[sequence] = sl
	f.inFlight++
	go func() {
		began := time.Now()
		body, err := f.hedged(f.life, s)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.inFlight--
		sl.body, sl.err = body, err
		if err == nil {
			f.took = append(f.took, time.Since(began))
			if len(f.took) > remembered {
				f.took = f.took[1:]
			}
		}
		close(sl.done)
		f.pump()
	}()
	return sl
}

// hedged reads s, asking again on a fresh connection when a read runs well past recent ones or fails, and keeps the first whole answer.
func (f *feed) hedged(ctx context.Context, s timeline.Segment) ([]byte, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type answer struct {
		body []byte
		err  error
	}
	answers := make(chan answer, attempts)
	asked, running := 0, 0
	ask := func() {
		asked++
		running++
		go func() {
			body, err := f.read(ctx, s)
			if err != nil {
				answers <- answer{err: err}
				return
			}
			defer func() { _ = body.Close() }()
			b, err := io.ReadAll(body)
			answers <- answer{b, err}
		}()
	}
	ask()
	hedge := time.NewTimer(f.hedgeAfter())
	defer hedge.Stop()
	var failure error
	for {
		select {
		case a := <-answers:
			running--
			if a.err == nil {
				if asked > 1 {
					slog.DebugContext(ctx, "a hedged segment read answered", "input", f.name, "uri", s.URI, "asked", asked)
				}
				return a.body, nil
			}
			failure = a.err
			// The origin's final no is the same on every connection.
			if refusal(a.err) != 0 || ctx.Err() != nil {
				return nil, a.err
			}
			if asked < attempts {
				ask()
			} else if running == 0 {
				return nil, failure
			}
		case <-hedge.C:
			if asked < attempts {
				slog.DebugContext(ctx, "a segment read is running slow; asking again on a fresh connection", "input", f.name, "uri", s.URI, "asked", asked)
				ask()
				hedge.Reset(f.hedgeAfter())
			}
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		}
	}
}

// hedgeAfter is twice the median of recent reads, within its floor and the feed's patience; with none yet, a default.
func (f *feed) hedgeAfter() time.Duration {
	f.mu.Lock()
	took := slices.Clone(f.took)
	f.mu.Unlock()
	if len(took) == 0 {
		return min(hedgeDefault, f.patience)
	}
	slices.Sort(took)
	return min(max(2*took[len(took)/2], hedgeFloor), f.patience)
}
