package kind

// Type clarifies context of timestamp, duration and remoteEndpoint in a span.
type Type string

// Available Type values
const (
	Undetermined Type = ""
	Client       Type = "CLIENT"
	Server       Type = "SERVER"
	Producer     Type = "PRODUCER"
	Consumer     Type = "CONSUMER"
)
