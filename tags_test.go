package zipkin

import "testing"

func TestTagNilSpan(t *testing.T) {
	TagError.Set(nil, "any value really")
}
