// Package nats fornece integração entre o OpenTelemetry e o NATS.
//
// Implementa a interface [go.opentelemetry.io/otel/propagation.TextMapCarrier]
// para mensagens NATS, permitindo propagar contexto de trace entre
// microsserviços via headers de mensagem — seguindo o padrão W3C TraceContext.
//
// # Como funciona a propagação
//
// O OpenTelemetry propaga o contexto de trace através de um "carrier" —
// um adaptador que sabe ler e escrever no envelope da mensagem.
// Este pacote fornece esse adaptador para o [github.com/nats-io/nats.go].
//
// # Publicando uma mensagem (inject)
//
// Ao publicar, injete o contexto do trace ativo nos headers da mensagem:
//
//	msg := &nats.Msg{Subject: "orders.created"}
//	carrier := natsotel.NewCarrier(msg)
//	propagator.Inject(ctx, carrier)
//	nc.PublishMsg(msg)
//
// # Consumindo uma mensagem (extract)
//
// Ao consumir, extraia o contexto dos headers para continuar o trace:
//
//	nc.Subscribe("orders.created", func(msg *nats.Msg) {
//	    carrier := natsotel.NewCarrier(msg)
//	    ctx := propagator.Extract(context.Background(), carrier)
//	    ctx, span := tracer.Start(ctx, "processar-pedido")
//	    defer span.End()
//	    // span é automaticamente filho do span do publicador
//	})
//
// # Segurança
//
// O Carrier inicializa o Header da mensagem automaticamente se for nil,
// garantindo que nenhuma operação cause panic por nil pointer.
package nats
