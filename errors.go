package zipkin

import "errors"

// common errors
var (
	ErrInvalidEndpoint = errors.New("requires valid local endpoint")
)
