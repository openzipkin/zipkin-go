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
	"testing"
)

func TestBaggageWhiteList(t *testing.T) {
	b := New("X-Request-Id", "some-header", "x-request-id")

	var items int
	b.IterateWhiteList(func(key string) {
		if key != "some-header" && key != "x-request-id" {
			t.Errorf("Unexpected whitelist item: %s", key)
		}
		items++
	})

	if items != 2 {
		t.Errorf("Unexpected whitelist count: want %d, have %d", 2, items)
	}

	items = 0
	b.Init().IterateWhiteList(func(key string) {
		if key != "some-header" && key != "x-request-id" {
			t.Errorf("Unexpected whitelist item: %s", key)
		}
		items++
	})

	if items != 2 {
		t.Errorf("Unexpected whitelist count: want %d, have %d", 2, items)
	}
}

func TestBaggageValues(t *testing.T) {
	b := New("X-Request-Id", "Some-Header")

	t.Run("AddHeader", func(t *testing.T) {
		if b.AddHeader("Invalid-Key", "Invalid-Key-Value") {
			t.Errorf("expected Invalid-Key to return false")
		}
		if !b.AddHeader("X-Request-Id", "X-Request-Id-Value") {
			t.Errorf("expected X-Request-Id to return true")
		}
		if !b.AddHeader("Some-Header", "Some-Header-Value1", "Some-Header-Value2") {
			t.Errorf("expected Some-Header to return true")
		}
		if !b.AddHeader("Some-Header", "Some-Header-Value3") {
			t.Errorf("expected Some-Header to return true")
		}
	})

	b.Init().IterateHeaders(func(key string, values []string) {
		t.Errorf("expected no header data to exist, have: key=%s values=%v", key, values)
	})

	t.Run("IterateHeaders", func(t *testing.T) {
		b.IterateHeaders(func(key string, have []string) {
			if strings.EqualFold(key, "x-request-id") {
				want := 1
				if len(have) != want {
					t.Errorf("expected different value count: want %d, have %d", want, len(have))
				}
				if have[0] != "X-Request-Id-Value" {
					t.Errorf("expected different value: want %s, have %s", "X-Request-Id-Value", have[0])
				}
				return
			}
			if strings.EqualFold(key, "some-header") {
				want := 3
				if len(have) != want {
					t.Errorf("expected different value count: want %d, have %d", want, len(have))
				}
				wantVal := "Some-Header-Value1"
				if have[0] != wantVal {
					t.Errorf("expected different value: want %s, have %s", wantVal, have[0])
				}
				wantVal = "Some-Header-Value2"
				if have[1] != wantVal {
					t.Errorf("expected different value: want %s, have %s", wantVal, have[1])
				}
				wantVal = "Some-Header-Value3"
				if have[2] != wantVal {
					t.Errorf("expected different value: want %s, have %s", wantVal, have[2])
				}
				return
			}
			t.Errorf("unexpected header key: %s", key)
		})
	})

	t.Run("DeleteHeader", func(t *testing.T) {
		if b.DeleteHeader("Invalid-Key") {
			t.Errorf("expected Invalid-Key to return false")
		}

		if !b.DeleteHeader("some-header") {
			t.Errorf("expected some-header to return true")
		}

		b.IterateHeaders(func(key string, have []string) {
			if strings.EqualFold(key, "x-request-id") {
				want := 1
				if len(have) != want {
					t.Errorf("expected different value count: want %d, have %d", want, len(have))
				}
				if have[0] != "X-Request-Id-Value" {
					t.Errorf("expected different value: want %s, have %s", "X-Request-Id-Value", have[0])
				}
				return
			}
			t.Errorf("unexpected header key: %s", key)
		})
	})

}
