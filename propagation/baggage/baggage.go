// Copyright 2021 The OpenZipkin Authors
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

package baggage

import (
	"strings"

	"github.com/openzipkin/zipkin-go/model"
)

var _ model.Baggage = (*baggage)(nil)

type baggage struct {
	wl map[string]bool
	c  map[string][]string
}

// New returns a new Baggage interface which is configured for the provided
// whitelisted headers.
func New(headers ...string) model.Baggage {
	b := &baggage{
		wl: make(map[string]bool),
		c:  make(map[string][]string),
	}
	for _, hdr := range headers {
		b.wl[strings.ToLower(hdr)] = true
	}
	return b
}

func (b *baggage) Init() model.Baggage {
	return &baggage{
		wl: b.wl,
		c:  make(map[string][]string),
	}
}

func (b *baggage) AddHeader(key string, val ...string) bool {
	if len(val) == 0 || !b.wl[key] {
		return false
	}
	// multiple values for a header is allowed
	b.c[key] = append(b.c[key], val...)

	return true
}

func (b *baggage) DeleteHeader(key string) bool {
	if !b.wl[key] {
		return false
	}
	delete(b.c, key)
	return true
}

func (b *baggage) IterateHeaders(f func(key string, vals []string)) {
	for k, v := range b.c {
		vals := make([]string, len(v))
		copy(vals, v)
		f(k, vals)
	}
}

func (b *baggage) IterateWhiteList(f func(key string)) {
	for k, _ := range b.wl {
		f(k)
	}
}
