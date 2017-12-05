package model

import (
	"fmt"
	"strconv"
)

// ID type
type ID uint64

// MarshalJSON serializes an ID type (SpanID, ParentSpanID) to HEX.
func (i ID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(
		"%016q", strconv.FormatUint(uint64(i), 16),
	)), nil
}
