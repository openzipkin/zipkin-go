package zipkin

import "time"

// FinishOption ...
type FinishOption func(s *span)

// FinishTime uses a finish time.
func FinishTime(t time.Time) FinishOption {
	return func(s *span) {
		s.Duration = t.Sub(s.Timestamp)
	}
}

// Duration uses a duration.
func Duration(d time.Duration) FinishOption {
	return func(s *span) {
		s.Duration = d
	}
}
