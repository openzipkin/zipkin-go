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

package zipkin

// Common Tag values
const (
	TagHTTPMethod       = "http.method"
	TagHTTPPath         = "http.path"
	TagHTTPUrl          = "http.url"
	TagHTTPRoute        = "http.route"
	TagHTTPStatusCode   = "http.status_code"
	TagHTTPRequestSize  = "http.request.size"
	TagHTTPResponseSize = "http.response.size"
	TagGRPCStatusCode   = "grpc.status_code"
	TagSQLQuery         = "sql.query"
	TagError            = "error"
)
