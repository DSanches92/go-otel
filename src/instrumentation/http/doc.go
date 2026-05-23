// Package http fornece middleware OpenTelemetry para aplicações HTTP em Go.
//
// Instrumenta automaticamente cada request com traces, atributos semânticos
// HTTP e propagação de contexto W3C TraceContext — sem exigir alterações
// nos handlers existentes.
//
// # Compatibilidade
//
// O middleware é baseado em [net/http] e compatível com qualquer framework
// que use http.Handler como abstração, incluindo chi, gorilla/mux e outros.
//
// # O que é instrumentado automaticamente
//
// Para cada request, o middleware cria um Span com:
//   - Nome:                    "MÉTODO /rota" (ex: "GET /orders")
//   - http.request.method:     método HTTP (GET, POST, ...)
//   - url.path:                caminho da URL (/orders/123)
//   - http.response.status_code: status code da resposta
//   - Status de erro:          marcado automaticamente para status >= 400
//
// # Uso com net/http
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/orders", handleOrders)
//
//	handler := httpgotel.NewMiddleware(provider)(mux)
//	http.ListenAndServe(":8080", handler)
//
// # Uso com chi
//
//	r := chi.NewRouter()
//	r.Use(httpgotel.NewMiddleware(provider))
//	r.Get("/orders", handleOrders)
//
// # Propagação de contexto
//
// O middleware extrai automaticamente o contexto W3C TraceContext e Baggage
// dos headers da request de entrada, permitindo que o trace se propague
// corretamente entre serviços em uma arquitetura de microsserviços.
package http
