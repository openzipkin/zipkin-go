package zipkin

import (
	"math/rand"
	"sync"
	"time"
)

var (
	seededIDGen = rand.New(rand.NewSource(time.Now().UnixNano()))
	// The golang rand generators are *not* intrinsically thread-safe.
	seededIDLock sync.Mutex
)

// IDGenerator interface
type IDGenerator interface {
	SpanID() SpanID
	TraceID() TraceID
}

type RandomID64 struct{}

func (r *RandomID64) TraceID() (id TraceID) {
	seededIDLock.Lock()
	id = TraceID{
		Low: uint64(seededIDGen.Int63()),
	}
	seededIDLock.Unlock()
	return
}

func (r *RandomID64) SpanID() (id SpanID) {
	seededIDLock.Lock()
	id = SpanID(seededIDGen.Int63())
	seededIDLock.Unlock()
	return
}

type RandomID128 struct{}

func (r *RandomID128) TraceID() (id TraceID) {
	seededIDLock.Lock()
	id = TraceID{
		High: uint64(seededIDGen.Int63()),
		Low:  uint64(seededIDGen.Int63()),
	}
	seededIDLock.Unlock()
	return
}

func (r *RandomID128) SpanID() (id SpanID) {
	seededIDLock.Lock()
	id = SpanID(seededIDGen.Int63())
	seededIDLock.Unlock()
	return
}

type TimestampedRandom struct{}

func (t *TimestampedRandom) TraceID() (id TraceID) {
	seededIDLock.Lock()
	id = TraceID{
		High: uint64(time.Now().UnixNano()),
		Low:  uint64(seededIDGen.Int63()),
	}
	seededIDLock.Unlock()
	return
}

func (t *TimestampedRandom) SpanID() (id SpanID) {
	seededIDLock.Lock()
	id = SpanID(seededIDGen.Int63())
	seededIDLock.Unlock()
	return
}
