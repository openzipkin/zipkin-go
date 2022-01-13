// Copyright 2022 The OpenZipkin Authors
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

import (
	"context"
	"testing"
)

func TestSpanOrNoopFromContext(t *testing.T) {
	var (
		ctx   = context.Background()
		tr, _ = NewTracer(nil, WithLocalEndpoint(nil))
		span  = tr.StartSpan("test")
	)

	if want, have := defaultNoopSpan, SpanOrNoopFromContext(ctx); want != have {
		t.Errorf("Invalid response want %+v, have %+v", want, have)
	}

	ctx = NewContext(ctx, span)

	if want, have := span, SpanOrNoopFromContext(ctx); want != have {
		t.Errorf("Invalid response want %+v, have %+v", want, have)
	}

}
