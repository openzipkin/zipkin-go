// Copyright 2019 The OpenZipkin Authors
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
package http

import (
	"math/rand"
	"time"
)

const (
	minDelay = 1 * time.Second
	maxDelay = 120 * time.Second
	factor   = 1.6
	jitter   = 0.2
)

func backoff(retries uint) time.Duration {
	min, max := float64(minDelay), float64(maxDelay)
	delay := min
	for ; delay < max && retries != 0; retries-- {
		delay *= factor
	}
	if delay > max {
		delay = max
	}
	delay *= 1 + jitter*(2*rand.Float64()-1)
	if delay < min {
		delay = min
	}
	return time.Duration(delay)
}
