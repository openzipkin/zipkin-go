// Copyright 2020 The OpenZipkin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zipkin

import "time"

// SpanCustomizer allows to safely customize a span without accesing its lifecycle
// methods
type SpanCustomizer interface {
	// Annotate adds a timed event to the Span.
	Annotate(time.Time, string)

	// Tag sets Tag with given key and value to the Span. If key already exists in
	// the Span the value will be overridden except for error tags where the first
	// value is persisted.
	Tag(string, string)
}

type shield struct {
	s Span
}

var _ SpanCustomizer = &shield{}

func (sc *shield) Annotate(t time.Time, value string) {
	sc.s.Annotate(t, value)
}

func (sc *shield) Tag(key string, value string) {
	sc.s.Tag(key, value)
}

// WrapWithSpanCustomizerShield wraps a span with the span customizer shield
func WrapWithSpanCustomizerShield(s Span) SpanCustomizer {
	return &shield{s}
}
