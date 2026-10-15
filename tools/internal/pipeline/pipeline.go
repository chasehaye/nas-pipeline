// Package pipeline tails fixm.raw into the accumulator, snapshotting and
// committing offsets on a timer. It owns the loop; the accumulator owns the model.
package pipeline

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/chasehaye/nas-pipeline/tools/internal/accumulator"
)

type Consumer interface {
	Fetch(ctx context.Context) (kafka.Message, error)
	Commit(ctx context.Context, msg kafka.Message) error
}

type Pipeline struct {
	consumer      Consumer
	acc           *accumulator.Accumulator
	snapshotPath  string
	snapshotEvery time.Duration
}

func New(consumer Consumer, acc *accumulator.Accumulator, snapshotPath string, snapshotEvery time.Duration) *Pipeline {
	return &Pipeline{consumer: consumer, acc: acc, snapshotPath: snapshotPath, snapshotEvery: snapshotEvery}
}

func (p *Pipeline) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Only the fetch loop writes last/haveLast; the snapshot goroutine reads them.
	var mu sync.Mutex
	var last kafka.Message
	var haveLast bool

	go func() {
		t := time.NewTicker(p.snapshotEvery)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := p.acc.SaveFile(p.snapshotPath); err != nil {
					slog.Error("snapshot write failed", "err", err)
				}
				mu.Lock()
				m, ok := last, haveLast
				mu.Unlock()
				if ok {
					if err := p.consumer.Commit(ctx, m); err != nil && ctx.Err() == nil {
						slog.Warn("offset commit failed", "err", err)
					}
				}
			}
		}
	}()

	for {
		msg, err := p.consumer.Fetch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			slog.Error("consume error", "err", err)
			continue
		}
		// unparseable messages are skipped; the census observes, it doesn't gate
		if err := p.acc.Observe(msg.Value); err != nil {
			slog.Warn("skipping unparseable message", "offset", msg.Offset, "err", err)
		}
		mu.Lock()
		last, haveLast = msg, true
		mu.Unlock()
	}

	slog.Info("shutting down, writing final snapshot", "path", p.snapshotPath)
	return p.acc.SaveFile(p.snapshotPath)
}
