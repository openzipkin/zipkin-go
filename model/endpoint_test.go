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

package model_test

import (
	"net"
	"testing"

	"github.com/openzipkin/zipkin-go/model"
)

func TestEmptyEndpoint(t *testing.T) {
	var e *model.Endpoint

	if want, have := true, e.Empty(); want != have {
		t.Errorf("Endpoint want %t, have %t", want, have)
	}

	e = &model.Endpoint{}

	if want, have := true, e.Empty(); want != have {
		t.Errorf("Endpoint want %t, have %t", want, have)
	}

	e = &model.Endpoint{
		IPv4: net.IPv4zero,
	}

	if want, have := false, e.Empty(); want != have {
		t.Errorf("Endpoint want %t, have %t", want, have)
	}

	e = &model.Endpoint{
		IPv6: net.IPv6zero,
	}

	if want, have := false, e.Empty(); want != have {
		t.Errorf("Endpoint want %t, have %t", want, have)
	}
}
