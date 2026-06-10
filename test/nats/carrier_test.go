package nats_test

import (
	"testing"

	natsotel "github.com/DSanches92/go-otel/src/nats"
	"github.com/nats-io/nats.go"
)

func newMessageWithHeader() *nats.Msg {
	msg := &nats.Msg{}
	msg.Header = make(nats.Header)
	return msg
}

func TestCarrier_Set(test *testing.T) {
	test.Run("deve inserir chave e valor no header da mensagem", func(test *testing.T) {
		msg := newMessageWithHeader()
		carrier := natsotel.NewCarrier(msg)

		carrier.Set("traceparent", "00-abc123-def456-01")

		got := carrier.Get("traceparent")
		if got != "00-abc123-def456-01" {
			test.Errorf("esperado '00-abc123-def456-01', obtido '%s'", got)
		}
	})

	test.Run("deve sobrescrever valor se a chave já existir", func(test *testing.T) {
		msg := newMessageWithHeader()
		msg.Header.Set("traceparent", "valor-antigo")
		carrier := natsotel.NewCarrier(msg)

		carrier.Set("traceparent", "valor-novo")

		got := carrier.Get("traceparent")
		if got != "valor-novo" {
			test.Errorf("esperado 'valor-novo', obtido '%s'", got)
		}
	})

	test.Run("não deve entrar em panic quando Header é nil", func(test *testing.T) {
		msg := &nats.Msg{} // Header nil intencionalmente
		carrier := natsotel.NewCarrier(msg)

		defer func() {
			if r := recover(); r != nil {
				test.Errorf("não esperado panic, obtido '%v'", r)
			}
		}()

		carrier.Set("traceparent", "00-abc123-def456-01")
	})
}

func TestCarrier_Get(test *testing.T) {
	test.Run("deve retornar o valor de uma chave existente", func(test *testing.T) {
		msg := newMessageWithHeader()
		carrier := natsotel.NewCarrier(msg)
		carrier.Set("traceparent", "00-abc123-def456-01")

		got := carrier.Get("traceparent")

		if got != "00-abc123-def456-01" {
			test.Errorf("esperado '00-abc123-def456-01', obtido '%s'", got)
		}
	})

	test.Run("deve retornar string vazia para chave inexistente", func(test *testing.T) {
		msg := newMessageWithHeader()
		carrier := natsotel.NewCarrier(msg)

		got := carrier.Get("chave-que-nao-existe")

		if got != "" {
			test.Errorf("esperado '', obtido '%s'", got)
		}
	})

	test.Run("deve ser case-insensitive ao buscar chave", func(test *testing.T) {
		msg := newMessageWithHeader()
		msg.Header.Set("Traceparent", "00-abc123-def456-01")
		carrier := natsotel.NewCarrier(msg)

		variantes := []string{"traceparent", "TRACEPARENT", "Traceparent", "trACEparent"}

		for _, chave := range variantes {
			test.Run(chave, func(test *testing.T) {
				got := carrier.Get(chave)
				if got != "00-abc123-def456-01" {
					test.Errorf("chave '%s': esperado '00-abc123-def456-01', obtido '%s'", chave, got)
				}
			})
		}
	})

	test.Run("não deve entrar em panic quando Header é nil", func(test *testing.T) {
		msg := &nats.Msg{} // Header nil intencionalmente
		carrier := natsotel.NewCarrier(msg)

		defer func() {
			if r := recover(); r != nil {
				test.Errorf("não esperado panic, obtido '%v'", r)
			}
		}()

		got := carrier.Get("traceparent")

		if got != "" {
			test.Errorf("esperado '' com header nil, obtido '%s'", got)
		}
	})
}

func TestCarrier_Keys(test *testing.T) {
	test.Run("deve retornar todas as chaves presentes no header", func(test *testing.T) {
		msg := newMessageWithHeader()
		msg.Header.Set("traceparent", "00-abc123-def456-01")
		msg.Header.Set("tracestate", "rojo=00f067")
		carrier := natsotel.NewCarrier(msg)

		keys := carrier.Keys()

		if len(keys) != 2 {
			test.Errorf("esperado 2 chaves, obtido %d", len(keys))
		}
	})

	test.Run("deve retornar slice vazio quando não há headers", func(test *testing.T) {
		msg := newMessageWithHeader()
		carrier := natsotel.NewCarrier(msg)

		keys := carrier.Keys()

		if len(keys) != 0 {
			test.Errorf("esperado 0 chaves, obtido %d", len(keys))
		}
	})

	test.Run("não deve entrar em panic quando Header é nil", func(test *testing.T) {
		msg := &nats.Msg{}
		carrier := natsotel.NewCarrier(msg)

		defer func() {
			if r := recover(); r != nil {
				test.Errorf("não esperado panic, obtido '%v'", r)
			}
		}()

		keys := carrier.Keys()

		if len(keys) != 0 {
			test.Errorf("esperado 0 chaves com header nil, obtido %d", len(keys))
		}
	})
}
