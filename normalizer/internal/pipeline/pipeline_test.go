package pipeline

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/segmentio/kafka-go"

	"github.com/chasehaye/nas-pipeline/normalizer/internal/metrics"
)

const validEnvelope = `<MessageCollection><message><flight timestamp="t"><flightIdentification aircraftIdentification="N1"/><flightStatus fdpsFlightStatus="ACTIVE"/><gufi>g1</gufi></flight></message></MessageCollection>`

const malformedEnvelope = `<MessageCollection><message><flight timestamp="t"`

type fakePublisher struct {
	msgs []kafka.Message
	err  error
}

func (f *fakePublisher) Publish(_ context.Context, msgs ...kafka.Message) error {
	if f.err != nil {
		return f.err
	}
	f.msgs = append(f.msgs, msgs...)
	return nil
}

type fakeDLQ struct {
	calls int
	err   error
}

func (f *fakeDLQ) Publish(context.Context, kafka.Message, string, string, string) error {
	f.calls++
	return f.err
}

// process one envelope through a Normalizer wired to the given fakes (no
// consumer needed; process never touches it).
func runOne(pub Publisher, dlq DLQPublisher, value string) *atomic.Bool {
	var failed atomic.Bool
	New(nil, pub, dlq, 1).process(
		context.Background(),
		kafka.Message{Value: []byte(value)},
		metrics.NewStats(),
		&failed,
	)
	return &failed
}

// Good envelope: flights published, nothing dead-lettered, commit not gated.
func TestProcess_ValidPublishes(t *testing.T) {
	pub, dlq := &fakePublisher{}, &fakeDLQ{}
	if runOne(pub, dlq, validEnvelope).Load() {
		t.Error("failed set on a successful publish")
	}
	if len(pub.msgs) != 1 {
		t.Fatalf("published %d messages, want 1", len(pub.msgs))
	}
	if dlq.calls != 0 {
		t.Errorf("dlq called %d times, want 0", dlq.calls)
	}
}

// Poison is dead-lettered and must not gate the batch.
func TestProcess_PoisonDeadLettered(t *testing.T) {
	pub, dlq := &fakePublisher{}, &fakeDLQ{}
	if runOne(pub, dlq, malformedEnvelope).Load() {
		t.Error("dead-lettered poison must not fail the batch")
	}
	if dlq.calls != 1 {
		t.Errorf("dlq calls = %d, want 1", dlq.calls)
	}
	if len(pub.msgs) != 0 {
		t.Errorf("published %d messages, want 0", len(pub.msgs))
	}
}

// Publish failure gates the commit (at-least-once: reprocess on restart).
func TestProcess_PublishFailureGates(t *testing.T) {
	pub := &fakePublisher{err: errors.New("broker down")}
	if !runOne(pub, &fakeDLQ{}, validEnvelope).Load() {
		t.Error("publish failure must set failed")
	}
}

// Failed dead-letter write also gates, so poison is retried, not dropped.
func TestProcess_DLQFailureGates(t *testing.T) {
	dlq := &fakeDLQ{err: errors.New("dlq down")}
	if !runOne(&fakePublisher{}, dlq, malformedEnvelope).Load() {
		t.Error("failed dead-letter write must set failed")
	}
}
