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

// IDGenerator interface
type IDGenerator interface {
	SpanID() SpanID
	TraceID() TraceID
}

// RandomID64 can generate 64 bit traceid's and 64 bit spanid's.
type RandomID64 struct{}

// TraceID implements IDGenerator
func (r *RandomID64) TraceID() (id TraceID) {
	seededIDLock.Lock()
	id = TraceID{
		Low: uint64(seededIDGen.Int63()),
	}
	seededIDLock.Unlock()
	return
}

// SpanID implements IDGenerator
func (r *RandomID64) SpanID() (id SpanID) {
	seededIDLock.Lock()
	id = SpanID(seededIDGen.Int63())
	seededIDLock.Unlock()
	return
}

// RandomID128 can generate 128 bit traceid's and 64 bit spanid's.
type RandomID128 struct{}

// TraceID implements IDGenerator
func (r *RandomID128) TraceID() (id TraceID) {
	seededIDLock.Lock()
	id = TraceID{
		High: uint64(seededIDGen.Int63()),
		Low:  uint64(seededIDGen.Int63()),
	}
	seededIDLock.Unlock()
	return
}

// SpanID implements IDGenerator
func (r *RandomID128) SpanID() (id SpanID) {
	seededIDLock.Lock()
	id = SpanID(seededIDGen.Int63())
	seededIDLock.Unlock()
	return
}

// TimestampedRandom can generate 128 bit time sortable traceid's and 64 bit
// spanid's.
type TimestampedRandom struct{}

// TraceID implements IDGenerator
func (t *TimestampedRandom) TraceID() (id TraceID) {
	seededIDLock.Lock()
	id = TraceID{
		High: uint64(time.Now().UnixNano()),
		Low:  uint64(seededIDGen.Int63()),
	}
	seededIDLock.Unlock()
	return
}

// SpanID implements IDGenerator
func (t *TimestampedRandom) SpanID() (id SpanID) {
	seededIDLock.Lock()
	id = SpanID(seededIDGen.Int63())
	seededIDLock.Unlock()
	return
}
