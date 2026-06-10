package nats

import (
	"net/textproto"

	"github.com/nats-io/nats.go"
)

// ---- Carrier

type Carrier struct {
	message *nats.Msg
}

func NewCarrier(message *nats.Msg) *Carrier {
	if message.Header == nil {
		message.Header = make(nats.Header)
	}

	return &Carrier{message: message}
}

// ---- Interface TextMapCarrier

func (carrier *Carrier) Get(key string) string {
	return carrier.message.Header.Get(textproto.CanonicalMIMEHeaderKey(key))
}

func (carrier *Carrier) Set(key, value string) {
	carrier.message.Header.Set(textproto.CanonicalMIMEHeaderKey(key), value)
}

func (carrier *Carrier) Keys() []string {
	keys := make([]string, 0, len(carrier.message.Header))

	for key := range carrier.message.Header {
		keys = append(keys, key)
	}

	return keys
}
