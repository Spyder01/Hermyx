package middleware

import (
	"testing"

	"github.com/valyala/fasthttp"
)

type markerMiddleware struct {
	name   string
	events *[]string
}

func (m *markerMiddleware) BeforeRequest(ctx *fasthttp.RequestCtx) error {
	*m.events = append(*m.events, m.name+":before")
	return nil
}

func (m *markerMiddleware) AfterResponse(ctx *fasthttp.RequestCtx) error {
	*m.events = append(*m.events, m.name+":after")
	return nil
}

func TestChainOrder(t *testing.T) {
	var events []string

	mw1 := &markerMiddleware{name: "mw1", events: &events}
	mw2 := &markerMiddleware{name: "mw2", events: &events}

	chain := NewChain(mw1, mw2)

	handler := chain.Handle(func(ctx *fasthttp.RequestCtx) {
		events = append(events, "handler")
	})

	ctx := &fasthttp.RequestCtx{}
	handler(ctx)

	expected := []string{
		"mw1:before",
		"mw2:before",
		"handler",
		"mw1:after",
		"mw2:after",
	}

	if len(events) != len(expected) {
		t.Fatalf("expected %d events, got %d", len(expected), len(events))
	}

	for i := range expected {
		if events[i] != expected[i] {
			t.Fatalf("at index %d: expected %s, got %s", i, expected[i], events[i])
		}
	}
}
