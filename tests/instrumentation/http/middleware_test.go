package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httpgotel "github.com/DSanches92/go-otel/src/instrumentation/http"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// ---- Helpers de teste

func newTracerProviderInMemory(test *testing.T) (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	test.Helper()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)

	return provider, recorder
}

func handlerWithStatus(status int) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		writer.WriteHeader(status)
	}
}

func executeRequest(
	test *testing.T,
	middleware func(http.Handler) http.Handler,
	method, path string,
	handler http.Handler,
) []sdktrace.ReadOnlySpan {
	test.Helper()

	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	return nil
}

// ---- Criação do Span

func TestMiddleware_Span(test *testing.T) {
	test.Run("deve criar um span para cada request", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		middleware := httpgotel.NewMiddleware(provider)

		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		rec := httptest.NewRecorder()

		middleware(handlerWithStatus(http.StatusOK)).ServeHTTP(rec, req)

		spans := recorder.Ended()
		if len(spans) != 1 {
			test.Errorf("esperado 1 span, obtido %d", len(spans))
		}
	})

	test.Run("deve nomear o span com método e rota", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		middleware := httpgotel.NewMiddleware(provider)

		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		rec := httptest.NewRecorder()

		middleware(handlerWithStatus(http.StatusOK)).ServeHTTP(rec, req)

		spans := recorder.Ended()
		if spans[0].Name() != "GET /orders" {
			test.Errorf("esperado 'GET /orders', obtido '%s'", spans[0].Name())
		}
	})
}

// ---- Atributos do Span

func TestMiddleware_Attributes(test *testing.T) {
	test.Run("deve registrar o método HTTP no span", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		middleware := httpgotel.NewMiddleware(provider)

		req := httptest.NewRequest(http.MethodPost, "/orders", nil)
		rec := httptest.NewRecorder()

		middleware(handlerWithStatus(http.StatusCreated)).ServeHTTP(rec, req)

		span := recorder.Ended()[0]
		assertAttribute(test, span, "http.request.method", "POST")
	})

	test.Run("deve registrar o path no span", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		middleware := httpgotel.NewMiddleware(provider)

		req := httptest.NewRequest(http.MethodGet, "/orders/123", nil)
		rec := httptest.NewRecorder()

		middleware(handlerWithStatus(http.StatusOK)).ServeHTTP(rec, req)

		span := recorder.Ended()[0]
		assertAttribute(test, span, "url.path", "/orders/123")
	})

	test.Run("deve registrar o status code no span", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		middleware := httpgotel.NewMiddleware(provider)

		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		rec := httptest.NewRecorder()

		middleware(handlerWithStatus(http.StatusOK)).ServeHTTP(rec, req)

		span := recorder.Ended()[0]
		assertAttributeInt(test, span, "http.response.status_code", http.StatusOK)
	})
}

// ---- Status de Erro no Span

func TestMiddleware_ErrorStatus(test *testing.T) {
	test.Run("deve marcar span como erro quando status >= 400", func(test *testing.T) {
		casos := []struct {
			status int
			nome   string
		}{
			{http.StatusBadRequest, "400 Bad Request"},
			{http.StatusUnauthorized, "401 Unauthorized"},
			{http.StatusNotFound, "404 Not Found"},
			{http.StatusInternalServerError, "500 Internal Server Error"},
			{http.StatusServiceUnavailable, "503 Service Unavailable"},
		}

		for _, caso := range casos {
			test.Run(caso.nome, func(test *testing.T) {
				provider, recorder := newTracerProviderInMemory(test)
				middleware := httpgotel.NewMiddleware(provider)

				req := httptest.NewRequest(http.MethodGet, "/orders", nil)
				rec := httptest.NewRecorder()

				middleware(handlerWithStatus(caso.status)).ServeHTTP(rec, req)

				span := recorder.Ended()[0]
				if span.Status().Code != codes.Error {
					test.Errorf("status %d: esperado span com código de erro, obtido '%s'",
						caso.status, span.Status().Code)
				}
			})
		}
	})

	test.Run("não deve marcar span como erro quando status < 400", func(test *testing.T) {
		casos := []struct {
			status int
			nome   string
		}{
			{http.StatusOK, "200 OK"},
			{http.StatusCreated, "201 Created"},
			{http.StatusNoContent, "204 No Content"},
			{http.StatusMovedPermanently, "301 Moved Permanently"},
		}

		for _, caso := range casos {
			test.Run(caso.nome, func(test *testing.T) {
				provider, recorder := newTracerProviderInMemory(test)
				middleware := httpgotel.NewMiddleware(provider)

				req := httptest.NewRequest(http.MethodGet, "/orders", nil)
				rec := httptest.NewRecorder()

				middleware(handlerWithStatus(caso.status)).ServeHTTP(rec, req)

				span := recorder.Ended()[0]
				if span.Status().Code == codes.Error {
					test.Errorf("status %d: não esperado span com código de erro", caso.status)
				}
			})
		}
	})
}

// ---- Propagação de Contexto

func TestMiddleware_Propagation(test *testing.T) {
	test.Run("deve propagar contexto de trace via headers W3C", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		middleware := httpgotel.NewMiddleware(provider)

		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
		rec := httptest.NewRecorder()

		middleware(handlerWithStatus(http.StatusOK)).ServeHTTP(rec, req)

		span := recorder.Ended()[0]
		traceID := span.SpanContext().TraceID().String()
		if traceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
			test.Errorf("esperado traceID '4bf92f3577b34da6a3ce929d0e0e4736', obtido '%s'", traceID)
		}
	})

	test.Run("deve criar novo trace quando não há contexto no header", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		middleware := httpgotel.NewMiddleware(provider)

		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		rec := httptest.NewRecorder()

		middleware(handlerWithStatus(http.StatusOK)).ServeHTTP(rec, req)

		span := recorder.Ended()[0]
		if !span.SpanContext().IsValid() {
			test.Error("esperado span com contexto válido")
		}
	})
}

// ---- Compatibilidade

func TestMiddleware_Compatibility(test *testing.T) {
	test.Run("deve ser compatível com net/http padrão", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		middleware := httpgotel.NewMiddleware(provider)

		mux := http.NewServeMux()
		mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()

		middleware(mux).ServeHTTP(rec, req)

		if len(recorder.Ended()) != 1 {
			test.Errorf("esperado 1 span com net/http, obtido %d", len(recorder.Ended()))
		}
	})

	test.Run("deve retornar handler que implementa http.Handler", func(test *testing.T) {
		provider, _ := newTracerProviderInMemory(test)
		middleware := httpgotel.NewMiddleware(provider)

		var _ http.Handler = middleware(handlerWithStatus(http.StatusOK))
	})
}

// ---- Helpers de asserção

func assertAttribute(test *testing.T, span sdktrace.ReadOnlySpan, chave, valorEsperado string) {
	test.Helper()

	for _, attr := range span.Attributes() {
		if string(attr.Key) == chave {
			if attr.Value.AsString() != valorEsperado {
				test.Errorf("atributo '%s': esperado '%s', obtido '%s'",
					chave, valorEsperado, attr.Value.AsString())
			}
			return
		}
	}

	test.Errorf("atributo '%s' não encontrado no span", chave)
}

func assertAttributeInt(test *testing.T, span sdktrace.ReadOnlySpan, chave string, valorEsperado int) {
	test.Helper()

	for _, attr := range span.Attributes() {
		if string(attr.Key) == chave {
			if attr.Value.AsInt64() != int64(valorEsperado) {
				test.Errorf("atributo '%s': esperado '%d', obtido '%d'",
					chave, valorEsperado, attr.Value.AsInt64())
			}
			return
		}
	}

	test.Errorf("atributo '%s' não encontrado no span", chave)
}

func ensurePresentAttribute(test *testing.T, span sdktrace.ReadOnlySpan, chave string) {
	test.Helper()

	for _, attr := range span.Attributes() {
		if string(attr.Key) == chave {
			return
		}
	}

	test.Errorf("atributo '%s' não encontrado no span", chave)
}

// evitarWarningDeUnusedFunction garante que o compilador não reclame
// de funções helper declaradas mas usadas apenas em testes futuros.
var _ = attribute.String
var _ = ensurePresentAttribute
