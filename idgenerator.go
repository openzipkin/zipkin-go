package zipkin

import (
	"math/rand"
	"sync"
	"time"
)

var (
	seededIDGen = rand.New(rand.NewSource(time.Now().UnixNano()))
	// NewSource returns a new pseudo-random Source seeded with the given value.
	// Unlike the default Source used by top-level functions, this source is not
	// safe for concurrent use by multiple goroutines. Hence the need for a mutex.
	seededIDLock sync.Mutex
)

// IDGenerator interface can be used to provide the Zipkin Tracer with custom
// implementations to generate Span and Trace IDs.
type IDGenerator interface {
	SpanID() ID       // Generates a new Span ID
	TraceID() TraceID // Generates a new Trace ID
}

// NewRandom64 returns an ID Generator which can generate 64 bit trace and span
// id's
func NewRandom64() IDGenerator {
	return &randomID64{}
}

// NewRandom128 returns an ID Generator which can generate 128 bit trace and 64
// bit span id's
func NewRandom128() IDGenerator {
	return &randomID128{}
}

// NewRandomTimestamped generates 128 bit time sortable traceid's and 64 bit
// spanid's.
func NewRandomTimestamped() IDGenerator {
	return &randomTimestamped{}
}

// randomID64 can generate 64 bit traceid's and 64 bit spanid's.
type randomID64 struct{}

func (r *randomID64) TraceID() (id TraceID) {
	seededIDLock.Lock()
	id = TraceID{
		Low: uint64(seededIDGen.Int63()),
	}
	seededIDLock.Unlock()
	return
}

func (r *randomID64) SpanID() (id ID) {
	seededIDLock.Lock()
	id = ID(seededIDGen.Int63())
	seededIDLock.Unlock()
	return
}

// randomID128 can generate 128 bit traceid's and 64 bit spanid's.
type randomID128 struct{}

func (r *randomID128) TraceID() (id TraceID) {
	seededIDLock.Lock()
	id = TraceID{
		High: uint64(seededIDGen.Int63()),
		Low:  uint64(seededIDGen.Int63()),
	}
	seededIDLock.Unlock()
	return
}

func (r *randomID128) SpanID() (id ID) {
	seededIDLock.Lock()
	id = ID(seededIDGen.Int63())
	seededIDLock.Unlock()
	return
}

// randomTimestamped can generate 128 bit time sortable traceid's and 64 bit
// spanid's.
type randomTimestamped struct{}

func (t *randomTimestamped) TraceID() (id TraceID) {
	seededIDLock.Lock()
	id = TraceID{
		High: uint64(time.Now().UnixNano()),
		Low:  uint64(seededIDGen.Int63()),
	}
	seededIDLock.Unlock()
	return
}

func (t *randomTimestamped) SpanID() (id ID) {
	seededIDLock.Lock()
	id = ID(seededIDGen.Int63())
	seededIDLock.Unlock()
	return
}
