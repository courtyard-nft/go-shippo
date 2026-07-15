package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// newRecordingProvider installs a TracerProvider that records spans in memory
// and returns the recorder plus a cleanup that restores the previous provider.
func newRecordingProvider(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })
	return sr
}

func TestClient_EmitsClientSpanForEachRequest(t *testing.T) {
	sr := newRecordingProvider(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	if _, err := newTestClient(server.URL).RetrieveAddress(context.Background(), "adcfdabb8b1946fdb44c4570e764c576"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("got %d spans; want 1", len(spans))
	}
	// Object-ID segment must be normalized to bound span-name cardinality.
	if name := spans[0].Name(); !strings.HasSuffix(name, "GET /addresses/{id}") {
		t.Errorf("span name = %q; want suffix %q", name, "GET /addresses/{id}")
	}
}

func TestClient_PropagatesTraceContextButNotBaggage(t *testing.T) {
	newRecordingProvider(t)

	var gotTraceparent, gotBaggage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceparent = r.Header.Get("traceparent")
		gotBaggage = r.Header.Get("baggage")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	ctx, parent := otel.Tracer("test").Start(context.Background(), "parent")
	defer parent.End()

	member, err := baggage.NewMember("internal", "secret")
	if err != nil {
		t.Fatalf("building baggage member: %v", err)
	}
	bag, err := baggage.New(member)
	if err != nil {
		t.Fatalf("building baggage: %v", err)
	}
	ctx = baggage.ContextWithBaggage(ctx, bag)

	if _, err := newTestClient(server.URL).RetrieveAddress(ctx, "obj123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotTraceparent == "" {
		t.Error("expected traceparent header to be propagated to Shippo, got none")
	}
	if gotBaggage != "" {
		t.Errorf("baggage must not be sent to a third party; got %q", gotBaggage)
	}
}

func TestClient_ClientSpanIsChildOfCallerSpan(t *testing.T) {
	sr := newRecordingProvider(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	ctx, parent := otel.Tracer("test").Start(context.Background(), "parent")

	if _, err := newTestClient(server.URL).RetrieveAddress(ctx, "obj123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parent.End()

	var client sdktrace.ReadOnlySpan
	for _, s := range sr.Ended() {
		if s.SpanKind().String() == "client" {
			client = s
		}
	}
	if client == nil {
		t.Fatal("no client span recorded")
	}
	if client.Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Errorf("client span parent = %v; want caller span %v", client.Parent().SpanID(), parent.SpanContext().SpanID())
	}
}
