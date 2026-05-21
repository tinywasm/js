package js_test

import (
	"testing"

	"github.com/tinywasm/context"
	"github.com/tinywasm/fetch"
	. "github.com/tinywasm/fmt"
	"github.com/tinywasm/js"
)

// Canonical spec for the runtime selection + JS composition exposed by
// tinywasm/js. These tests define what the implementation must satisfy
// once js/docs/PLAN.md stages 5-6 land.
//
// The wasm_exec.js content getters are intentionally NOT public — behavior
// is verified through the public composers (PageBootstrap / ServiceWorker),
// which inline the runtime selected via SetRuntime.

// Signatures that distinguish Go's wasm_exec.js (declared inline; the public
// API does not export signature lists).
var goRuntimeSignatures = []string{
	"runtime.scheduleTimeoutEvent",
	"runtime.clearTimeoutEvent",
	"runtime.wasmExit",
}

// Signatures that distinguish TinyGo's wasm_exec.js.
var tinyGoRuntimeSignatures = []string{
	"runtime.sleepTicks",
	"runtime.ticks",
	"gojs",
}

func TestPageBootstrap_IsBundleScript(t *testing.T) {
	s := js.PageBootstrap()
	if s.Name != "" {
		t.Errorf("PageBootstrap().Name = %q, want \"\" (goes into /script.js bundle)", s.Name)
	}
	if s.Content == "" {
		t.Fatal("PageBootstrap().Content is empty")
	}
}

func TestPageBootstrap_ReferencesClientWasm(t *testing.T) {
	c := js.PageBootstrap().Content
	if !Contains(c, "WebAssembly.instantiateStreaming") {
		t.Error("PageBootstrap() missing WebAssembly.instantiateStreaming")
	}
	if !Contains(c, "/client.wasm") {
		t.Error("PageBootstrap() must fetch /client.wasm")
	}
}

func TestPageBootstrap_InlinesRuntimePerSetRuntime(t *testing.T) {
	t.Cleanup(func() { js.SetRuntime(js.RuntimeGo) })

	js.SetRuntime(js.RuntimeTinyGo)
	tiny := js.PageBootstrap().Content
	for _, sig := range tinyGoRuntimeSignatures {
		if !Contains(tiny, sig) {
			t.Errorf("after SetRuntime(TinyGo), bootstrap missing TinyGo signature %q", sig)
		}
	}
	for _, sig := range goRuntimeSignatures {
		if Contains(tiny, sig) {
			t.Errorf("after SetRuntime(TinyGo), bootstrap unexpectedly contains Go signature %q", sig)
		}
	}

	js.SetRuntime(js.RuntimeGo)
	go_ := js.PageBootstrap().Content
	for _, sig := range goRuntimeSignatures {
		if !Contains(go_, sig) {
			t.Errorf("after SetRuntime(Go), bootstrap missing Go signature %q", sig)
		}
	}
}

func TestServiceWorker_FixedName(t *testing.T) {
	s := js.ServiceWorker(nopServiceWorkerHandler{})
	if s.Name != "sw.js" {
		t.Errorf("ServiceWorker().Name = %q, want \"sw.js\"", s.Name)
	}
	if s.Content == "" {
		t.Fatal("ServiceWorker().Content is empty")
	}
}

func TestServiceWorker_ContainsHooks(t *testing.T) {
	c := js.ServiceWorker(nopServiceWorkerHandler{}).Content
	for _, hook := range []string{"install", "activate", "fetch"} {
		if !Contains(c, hook) {
			t.Errorf("ServiceWorker() shim missing %q listener", hook)
		}
	}
}

func TestWebWorker_UsesGivenName(t *testing.T) {
	s := js.WebWorker("parser.worker.js", nopWebWorkerHandler{})
	if s.Name != "parser.worker.js" {
		t.Errorf("WebWorker().Name = %q, want \"parser.worker.js\"", s.Name)
	}
}

type nopServiceWorkerHandler struct{}

func (nopServiceWorkerHandler) OnInstall(_ context.Context) error  { return nil }
func (nopServiceWorkerHandler) OnActivate(_ context.Context) error { return nil }
func (nopServiceWorkerHandler) OnFetch(_ context.Context, _ *js.Request) (*fetch.Response, error) {
	return nil, nil
}

type nopWebWorkerHandler struct{}

func (nopWebWorkerHandler) OnMessage(_ context.Context, _ *js.Message) (*js.Message, error) {
	return nil, nil
}
