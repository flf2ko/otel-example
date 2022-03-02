package tracer

import (
	"context"
	"math/rand"
	"sync"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type cidIDGenerator struct {
	sync.Mutex
	randSource *rand.Rand
}

var _ tracesdk.IDGenerator = &cidIDGenerator{}

func GetCIDIDGenerator() tracesdk.IDGenerator {
	return &cidIDGenerator{}
}

// NewSpanID returns a non-zero span ID from a randomly-chosen sequence.
func (gen *cidIDGenerator) NewSpanID(ctx context.Context, traceID trace.TraceID) trace.SpanID {
	gen.Lock()
	defer gen.Unlock()
	sid := trace.SpanID{}
	gen.randSource.Read(sid[:])
	return sid
}

// NewIDs returns a CID as trace ID and a non-zero span ID from a
// randomly-chosen sequence.
func (gen *cidIDGenerator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	gen.Lock()
	defer gen.Unlock()
	tid := trace.TraceID{}
	sid := trace.SpanID{}
	gen.randSource.Read(sid[:])
	return tid, sid
}
