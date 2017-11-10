package zipkin

import (
	"fmt"
	"strconv"
)

// ID type
type ID uint64

// MarshalJSON serializes SpanID to HEX.
func (i ID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(
		"%016q", strconv.FormatUint(uint64(i), 16),
	)), nil
}
