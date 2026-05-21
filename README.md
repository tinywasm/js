# tinywasm/js
<img src="docs/img/badges.svg">

Typed layer for Service Workers and Web Workers in TinyWASM.

> **Write Go. The framework generates the JS shim.**

## Overview

`tinywasm/js` is the **only JS API** in the TinyWASM framework — mirroring what `tinywasm/css` does for stylesheets. SSR modules call typed constructors returning `*Script` values with final JS content; `assetmin` writes them to disk without additional coordination.

## API

### Script (v1 escape hatch)

The core `Script` type is available for arbitrary JS snippets:

```go
type Script struct {
    Name    string // Empty → bundled into /script.js. Non-empty → standalone /public/<Name>.
    Content string // Raw JavaScript.
}
```

### Runtime configuration (call once at boot)

```go
js.SetRuntime(js.RuntimeGo)    // Standard Go compiler
js.SetRuntime(js.RuntimeTinyGo) // TinyGo compiler (smaller binaries)
```

`tinywasm/app` calls this automatically when it detects the compiler mode — **user modules never call it directly**.

### Typed constructors (recommended)

```go
// Bundles wasm_exec.js + WASM bootstrap into /script.js.
js.PageBootstrap() *Script

// Generates /sw.js with wasm_exec.js + SW event listeners inlined.
js.ServiceWorker(handler ServiceWorkerHandler) *Script

// Generates /<name> with wasm_exec.js + message listener inlined.
js.WebWorker(name string, handler WebWorkerHandler) *Script
```

### Handler interfaces

Implement in Go — the shim bridges browser events to your methods:

```go
import (
    "github.com/tinywasm/context" // stdlib context is vetoed in WASM
    "github.com/tinywasm/fetch"
)

type ServiceWorkerHandler interface {
    OnInstall(ctx context.Context) error
    OnActivate(ctx context.Context) error
    OnFetch(ctx context.Context, req *js.Request) (*fetch.Response, error)
}

type WebWorkerHandler interface {
    OnMessage(ctx context.Context, msg *js.Message) (*js.Message, error)
}
```

### Event types

```go
// Request is the inbound FetchEvent intercepted by the SW.
type Request struct {
    URL     string
    Method  string
    Headers []fetch.Header // {Key, Value string} — no maps
    Body    []byte
}

// Message is the postMessage payload for Web Workers.
type Message struct {
    Data []byte
}
```

## Service Worker example (PWA)

```go
package mymodule

import (
    "github.com/tinywasm/context"
    "github.com/tinywasm/fetch"
    "github.com/tinywasm/js"
)

type CachingSW struct{}

func (sw *CachingSW) OnInstall(ctx context.Context) error  { return nil }
func (sw *CachingSW) OnActivate(ctx context.Context) error { return nil }

func (sw *CachingSW) OnFetch(ctx context.Context, req *js.Request) (*fetch.Response, error) {
    // Only intercept API routes; fall through for statics.
    body := []byte(`{"cached": true}`)
    return fetch.NewResponse(200,
        []fetch.Header{{Key: "Content-Type", Value: "application/json"}},
        body,
    ), nil
}

func (m Module) RenderJS() []*js.Script {
    return []*js.Script{
        js.PageBootstrap(),          // wasm_exec + bootstrap → bundled in /script.js
        js.ServiceWorker(&CachingSW{}), // full shim → /sw.js
    }
}
```

## Web Worker example

```go
type ParserWorker struct{}

func (w *ParserWorker) OnMessage(ctx context.Context, msg *js.Message) (*js.Message, error) {
    // Process msg.Data ([]byte) and return result.
    result := process(msg.Data)
    return &js.Message{Data: result}, nil
}

func (m Module) RenderJS() []*js.Script {
    return []*js.Script{
        js.WebWorker("parser.worker.js", &ParserWorker{}),
    }
}
```

## Escape hatch

For raw JS snippets (analytics, polyfills, init scripts):

```go
// Bundled into /script.js
&js.Script{Content: "console.log('init')"}

// Written as standalone file
&js.Script{Name: "analytics.js", Content: analyticsJS}
```

## Stdlib constraints

`tinywasm/js` compiles to WASM — the Go stdlib is **vetoed** to keep binary size minimal.

| Stdlib (prohibited) | Replacement |
|---|---|
| `context` | `github.com/tinywasm/context` |
| `fmt`, `errors`, `strings`, `strconv`, `path` | `github.com/tinywasm/fmt` |
| `encoding/json` | `github.com/tinywasm/json` |
| `time` | `github.com/tinywasm/time` |
| `map[string]string` (headers) | `[]fetch.Header` |
