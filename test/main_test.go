package gotel_test

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreCurrent(),
		goleak.IgnoreTopFunction(
			"go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue",
		),
	)
}
