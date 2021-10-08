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

package http

import "net/http"

// RequestSamplerFunc can be implemented for client and/or server side sampling decisions that can override the existing
// upstream sampling decision. If the implementation returns nil, the existing sampling decision stays as is.
type RequestSamplerFunc func(r *http.Request) *bool

// Sample is a convenience function that returns a pointer to a boolean true. Use this for RequestSamplerFuncs when
// wanting the RequestSampler to override the sampling decision to yes.
func Sample() *bool {
	sample := true
	return &sample
}

// Discard is a convenience function that returns a pointer to a boolean false. Use this for RequestSamplerFuncs when
// wanting the RequestSampler to override the sampling decision to no.
func Discard() *bool {
	sample := false
	return &sample
}
