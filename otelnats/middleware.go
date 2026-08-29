package otelnats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

func NewTracer(tracer trace.Tracer) *Tracer {
	return &Tracer{
		tracer: tracer,
		propagator: propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	}
}

func (t *Tracer) Subscribe(nc *nats.Conn, subject string, handler func(context.Context, *nats.Msg)) (*nats.Subscription, error) {
	return nc.Subscribe(subject, func(msg *nats.Msg) {
		carrier := NewCarrier(msg)
		ctx := t.propagator.Extract(context.Background(), carrier)
		ctx, span := t.tracer.Start(ctx, "SUBSCRIBE "+subject)
		defer span.End()

		handler(ctx, msg)
	})
}

func (t *Tracer) Publish(ctx context.Context, nc *nats.Conn, subject string, data []byte) error {
	ctx, span := t.tracer.Start(ctx, "PUBLISH "+subject)
	defer span.End()

	msg := &nats.Msg{Subject: subject, Data: data}
	carrier := NewCarrier(msg)
	t.propagator.Inject(ctx, carrier)

	err := nc.PublishMsg(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (t *Tracer) QueueSubscribe(nc *nats.Conn, subject, queue string, handler func(context.Context, *nats.Msg)) (*nats.Subscription, error) {
	return nc.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		carrier := NewCarrier(msg)
		ctx := t.propagator.Extract(context.Background(), carrier)
		ctx, span := t.tracer.Start(ctx, fmt.Sprintf("SUBSCRIBE %s [%s]", subject, queue))
		defer span.End()

		handler(ctx, msg)
	})
}
