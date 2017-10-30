package zipkin

// Kind clarifies context of timestamp, duration and remoteEndpoint in a span.
type Kind string

// Kind values available
const (
	Client   Kind = "CLIENT"
	Server   Kind = "SERVER"
	Producer Kind = "PRODUCER"
	Consumer Kind = "CONSUMER"
)
