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

// Package baggage holds a Baggage propagation implementation based on
// explicit allowList semantics.
package baggage

import (
	"strings"

	"github.com/openzipkin/zipkin-go/middleware"
	"github.com/openzipkin/zipkin-go/model"
)

var (
	_ middleware.BaggageHandler = (*baggage)(nil)
	_ model.BaggageFields       = (*baggage)(nil)
)

type baggage struct {
	// registry holds our registry of allowed fields to propagate
	registry map[string]struct{}
	// fields holds the retrieved key-values pairs to propagate
	fields map[string][]string
}

// New returns a new Baggage interface which is configured to propagate the
// registered fields.
func New(keys ...string) middleware.BaggageHandler {
	b := &baggage{
		registry: make(map[string]struct{}),
	}
	for _, key := range keys {
		b.registry[strings.ToLower(key)] = struct{}{}
	}
	return b
}

// New is called by server middlewares and returns a fresh initialized
// baggage implementation.
func (b *baggage) New() model.BaggageFields {
	return &baggage{
		registry: b.registry,
		fields:   make(map[string][]string),
	}
}

func (b *baggage) Get(key string) []string {
	return b.fields[strings.ToLower(key)]
}

func (b *baggage) Add(key string, values ...string) bool {
	if len(values) == 0 {
		return false
	}
	key = strings.ToLower(key)
	if _, ok := b.registry[key]; !ok {
		return false
	}
	// multiple values for a header is allowed
	b.fields[key] = append(b.fields[key], values...)

	return true
}

func (b *baggage) Set(key string, values ...string) bool {
	if len(values) == 0 {
		return false
	}
	key = strings.ToLower(key)
	if _, ok := b.registry[key]; !ok {
		return false
	}
	b.fields[key] = values

	return true
}

func (b *baggage) Delete(key string) bool {
	key = strings.ToLower(key)
	if _, ok := b.registry[key]; !ok {
		return false
	}
	for k := range b.fields {
		if key == k {
			delete(b.fields, k)
		}
	}
	return true
}

func (b *baggage) Iterate(f func(key string, values []string)) {
	for key, v := range b.fields {
		values := make([]string, len(v))
		copy(values, v)
		f(key, values)
	}
}

func (b *baggage) IterateKeys(f func(key string)) {
	for key := range b.registry {
		f(key)
	}
}
