package otelhttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DSanches92/go-otel/otelhttp"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTransport_Span(test *testing.T) {
	test.Run("deve criar span para request de saída", func(test *testing.T) {
		recorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(recorder),
		)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		transport := otelhttp.NewTransport(provider)
		client := &http.Client{Transport: transport}

		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			test.Fatalf("não esperado erro, obtido '%s'", err)
		}
		resp.Body.Close()

		spans := recorder.Ended()
		if len(spans) != 1 {
			test.Fatalf("esperado 1 span, obtido %d", len(spans))
		}

		if spans[0].Name()[:4] != "GET " {
			test.Errorf("esperado span começando com 'GET ', obtido '%s'", spans[0].Name())
		}
	})

	test.Run("deve propagar traceparent no header", func(test *testing.T) {
		recorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(recorder),
		)

		var capturedTraceparent string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedTraceparent = r.Header.Get("traceparent")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		transport := otelhttp.NewTransport(provider)
		client := &http.Client{Transport: transport}

		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			test.Fatalf("não esperado erro, obtido '%s'", err)
		}
		resp.Body.Close()

		spans := recorder.Ended()
		if len(spans) != 1 {
			test.Fatalf("esperado 1 span, obtido %d", len(spans))
		}

		if capturedTraceparent == "" {
			test.Error("esperado header traceparent, obtido vazio")
		}
	})

	test.Run("deve marcar span como erro quando status >= 400", func(test *testing.T) {
		recorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(recorder),
		)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		transport := otelhttp.NewTransport(provider)
		client := &http.Client{Transport: transport}

		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			test.Fatalf("não esperado erro, obtido '%s'", err)
		}
		resp.Body.Close()

		spans := recorder.Ended()
		if len(spans) != 1 {
			test.Fatalf("esperado 1 span, obtido %d", len(spans))
		}

		if spans[0].Status().Code != codes.Error {
			test.Errorf("esperado status Error, obtido %v", spans[0].Status().Code)
		}
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestTransport_WithRoundTripper(test *testing.T) {
	test.Run("deve usar RoundTripper customizado", func(test *testing.T) {
		recorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(recorder),
		)

		called := false
		customRT := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			called = true
			return http.DefaultTransport.RoundTrip(req)
		})

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		transport := otelhttp.NewTransport(provider,
			otelhttp.WithRoundTripper(customRT),
		)
		client := &http.Client{Transport: transport}

		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			test.Fatalf("não esperado erro, obtido '%s'", err)
		}
		resp.Body.Close()

		if !called {
			test.Error("esperado RoundTripper customizado ser chamado")
		}
	})
}
