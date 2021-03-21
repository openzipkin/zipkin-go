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

package b3

import (
	"testing"

	"github.com/openzipkin/zipkin-go/model"
)

func TestParseHeaderSuccess(t *testing.T) {
	testCases := []struct {
		header          string
		expectedContext *model.SpanContext
		expectedErr     error
	}{
		{"d", &model.SpanContext{Debug: true}, nil},
		{"1", &model.SpanContext{Sampled: samplingVal(true)}, nil},
		{"000000000000007b00000000000001c8-000000000000007b", &model.SpanContext{TraceID: model.TraceID{High: 123, Low: 456}, ID: model.ID(123)}, nil},
		{"000000000000007b00000000000001c8-000000000000007b-0", &model.SpanContext{TraceID: model.TraceID{High: 123, Low: 456}, ID: model.ID(123), Sampled: samplingVal(false)}, nil},
		{"000000000000007b00000000000001c8-000000000000007b-1-00000000000001c8", &model.SpanContext{TraceID: model.TraceID{High: 123, Low: 456}, ID: model.ID(123), ParentID: parentIDVal(456), Sampled: samplingVal(true)}, nil},
		{"", nil, ErrEmptyContext},
		{"80f198ee56343ba864fe8b2a57d3eff7-e457b5a2e4d86bd1-1-05e3ac9a4f6e3b90", &model.SpanContext{TraceID: model.TraceID{High: 9291375655657946024, Low: 7277407061855694839}, ID: model.ID(16453819474850114513), ParentID: parentIDVal(424372568660523920), Sampled: samplingVal(true)}, nil},
		{"d4c3c787ce202dc5-77c6a763a5a72544-0-6a3211d95bed2c99", &model.SpanContext{TraceID: model.TraceID{High: 0, Low: 15331316942592028101}, ID: model.ID(8630769782324929860), ParentID: parentIDVal(7652198342103739545), Sampled: samplingVal(false)}, nil},
	}

	for _, testCase := range testCases {
		actualContext, actualErr := ParseSingleHeader(testCase.header)
		if testCase.expectedContext != nil {
			if actualErr != nil {
				t.Fatalf("unexpected error for header %q: %s", testCase.header, actualErr.Error())
			}
			if !(actualContext.TraceID == testCase.expectedContext.TraceID &&
				actualContext.ID == testCase.expectedContext.ID &&
				((actualContext.ParentID == nil && testCase.expectedContext.ParentID == nil) ||
					*actualContext.ParentID == *testCase.expectedContext.ParentID) &&
				((actualContext.Sampled == nil && testCase.expectedContext.Sampled == nil) ||
					*actualContext.Sampled == *testCase.expectedContext.Sampled) &&
				actualContext.Debug == testCase.expectedContext.Debug) {
				t.Fatalf("unexpected context for header %q, want: %+v, have: %+v", testCase.header, *testCase.expectedContext, *actualContext)
			}
		}

		if want, have := actualErr, testCase.expectedErr; want != have {
			if testCase.expectedErr == nil {
				t.Fatalf("unexpected error for header %q: %v", testCase.header, have)
			} else {
				t.Fatalf("unexpected error for header %q, want: %v, have: %v", testCase.header, want, have)
			}
		}
	}
}

func TestParseHeaderFails(t *testing.T) {
	testCases := []struct {
		header      string
		expectedErr error
	}{
		{"a", ErrInvalidSampledByte},
		{"3", ErrInvalidSampledByte},
		{"000000000000007b", ErrInvalidScope},
		{"000000000000007b00000000000001c8", ErrInvalidScope},
		{"000000000000007b00000000000001c8-000000000000007b-", ErrInvalidSampledByte},
		{"000000000000007b00000000000001c8-000000000000007b-3", ErrInvalidSampledByte},
		{"000000000000007b00000000000001c8-000000000000007b-00000000000001c8", ErrInvalidScopeParentSingle},
		{"000000000000007b00000000000001c8-000000000000007b-1-00000000000001c", ErrInvalidParentSpanIDValue},
		{"", ErrEmptyContext},
	}

	for _, testCase := range testCases {
		_, actualErr := ParseSingleHeader(testCase.header)
		if want, have := testCase.expectedErr, actualErr; want != have {
			t.Fatalf("unexpected error for header %q, want: %q, have: %q", testCase.header, want, have)
		}
	}
}

func TestBuildHeader(t *testing.T) {
	testCases := []struct {
		context        model.SpanContext
		expectedHeader string
	}{
		{model.SpanContext{ID: model.ID(123)}, ""},
		{model.SpanContext{Debug: true}, "d"},
		{model.SpanContext{Sampled: samplingVal(true)}, "1"},
		{model.SpanContext{TraceID: model.TraceID{High: 123, Low: 456}, ID: model.ID(123)}, "000000000000007b00000000000001c8-000000000000007b"},
		{model.SpanContext{TraceID: model.TraceID{High: 123, Low: 456}, ID: model.ID(123), Sampled: samplingVal(false)}, "000000000000007b00000000000001c8-000000000000007b-0"},
		{model.SpanContext{
			TraceID:  model.TraceID{High: 123, Low: 456},
			ID:       model.ID(123),
			ParentID: parentIDVal(456),
			Sampled:  samplingVal(false),
		}, "000000000000007b00000000000001c8-000000000000007b-0-00000000000001c8"},
	}

	for _, testCase := range testCases {
		actualHeader := BuildSingleHeader(testCase.context)
		if want, have := actualHeader, testCase.expectedHeader; want != have {
			t.Fatalf("unexpected header value, want: %s, have %s", want, have)
		}
	}
}

func parentIDVal(value int) *model.ID {
	parentIDVal := model.ID(value)
	return &parentIDVal
}

func samplingVal(value bool) *bool {
	return &value
}
