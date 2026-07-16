package nats_test

import (
	"context"
	"testing"
	"time"

	natsotel "github.com/DSanches92/go-otel/internal/nats"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	natsserver.DefaultTestOptions.Port = -1
}

func runNATSServer() (*server.Server, *nats.Conn) {
	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	srv := natsserver.RunServer(&opts)

	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		panic(err)
	}

	return srv, nc
}

func TestTracer_Subscribe(test *testing.T) {
	test.Run("deve criar span ao receber mensagem", func(test *testing.T) {
		recorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(recorder),
		)

		srv, nc := runNATSServer()
		defer srv.Shutdown()
		defer nc.Close()

		nt := natsotel.NewTracer(provider.Tracer("test"))
		received := make(chan struct{}, 1)

		sub, err := nt.Subscribe(nc, "orders.test", func(ctx context.Context, msg *nats.Msg) {
			span := trace.SpanFromContext(ctx)
			if !span.SpanContext().IsValid() {
				test.Error("esperado span válido no contexto")
			}
			received <- struct{}{}
		})
		if err != nil {
			test.Fatalf("subscribe error: %s", err)
		}
		defer sub.Unsubscribe()

		nc.Publish("orders.test", []byte("hello"))
		nc.Flush()

		select {
		case <-received:
		case <-time.After(time.Second):
			test.Fatal("timeout aguardando mensagem")
		}

		time.Sleep(100 * time.Millisecond)
		spans := recorder.Ended()
		if len(spans) == 0 {
			test.Fatal("esperado ao menos 1 span, obtido 0")
		}
		if spans[0].Name() != "SUBSCRIBE orders.test" {
			test.Errorf("esperado 'SUBSCRIBE orders.test', obtido '%s'", spans[0].Name())
		}
	})

	test.Run("deve propagar contexto entre publish e subscribe", func(test *testing.T) {
		recorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(recorder),
		)

		srv, nc := runNATSServer()
		defer srv.Shutdown()
		defer nc.Close()

		nt := natsotel.NewTracer(provider.Tracer("test"))
		received := make(chan struct{}, 1)
		var subscriberTraceID string

		sub, err := nt.Subscribe(nc, "orders.propagate", func(ctx context.Context, msg *nats.Msg) {
			span := trace.SpanFromContext(ctx)
			subscriberTraceID = span.SpanContext().TraceID().String()
			received <- struct{}{}
		})
		if err != nil {
			test.Fatalf("subscribe error: %s", err)
		}
		defer sub.Unsubscribe()

		parentCtx, parentSpan := provider.Tracer("test").Start(context.Background(), "parent")
		nt.Publish(parentCtx, nc, "orders.propagate", []byte("hello"))
		parentSpan.End()
		nc.Flush()

		select {
		case <-received:
		case <-time.After(time.Second):
			test.Fatal("timeout aguardando mensagem")
		}

		parentTraceID := parentSpan.SpanContext().TraceID().String()
		if subscriberTraceID != parentTraceID {
			test.Errorf("TraceID do subscriber '%s' difere do publisher '%s'",
				subscriberTraceID, parentTraceID)
		}
	})
}

func TestTracer_Publish(test *testing.T) {
	test.Run("deve criar span ao publicar mensagem", func(test *testing.T) {
		recorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(recorder),
		)

		srv, nc := runNATSServer()
		defer srv.Shutdown()
		defer nc.Close()

		nt := natsotel.NewTracer(provider.Tracer("test"))

		nc.Subscribe("orders.publish", func(msg *nats.Msg) {})
		nc.Flush()
		time.Sleep(50 * time.Millisecond)

		err := nt.Publish(context.Background(), nc, "orders.publish", []byte("hello"))
		if err != nil {
			test.Fatalf("não esperado erro, obtido '%s'", err)
		}

		time.Sleep(100 * time.Millisecond)
		spans := recorder.Ended()
		if len(spans) == 0 {
			test.Fatal("esperado ao menos 1 span, obtido 0")
		}
		if spans[0].Name() != "PUBLISH orders.publish" {
			test.Errorf("esperado 'PUBLISH orders.publish', obtido '%s'", spans[0].Name())
		}
	})
}
