# tinywasm/js

Typed layer for Service Workers and Web Workers in TinyWASM.

## Purpose

Allows SSR modules to declare Service Workers and Web Workers by writing **only typed Go**. `tinywasm/js` provides a minimal JS shim to bridge browser events (FetchEvent, MessageEvent) to Go handlers.

## API

### Script

The core `Script` type represents JS code emitted by a module.

```go
type Script struct {
    Name    string // Filename for standalone files (e.g., "sw.js"). Empty for bundling.
    Content string // Raw JavaScript code.
}
```

### Constructors

- `PageBootstrap()`: Main entrypoint for the page.
- `ServiceWorker(handler ServiceWorkerHandler)`: Registers a service worker.
- `WebWorker(name string, handler WebWorkerHandler)`: Registers a web worker.

### Handlers

Implement these interfaces to handle events in Go:

```go
type ServiceWorkerHandler interface {
    OnInstall(ctx context.Context) error
    OnActivate(ctx context.Context) error
    OnFetch(ctx context.Context, req *js.Request) (*fetch.Response, error)
}

type WebWorkerHandler interface {
    OnMessage(ctx context.Context, msg *js.Message) (*js.Message, error)
}
```

## Usage Example

```go
package mymodule

import (
    "github.com/tinywasm/js"
    "github.com/tinywasm/context"
    "github.com/tinywasm/fetch"
)

type MySW struct{}

func (h *MySW) OnFetch(ctx context.Context, req *js.Request) (*fetch.Response, error) {
    // Custom logic in Go
    return fetch.NewResponse(200, nil, []byte("Hello from SW")), nil
}

// In your module's RenderJS:
func (m Module) RenderJS() []*js.Script {
    return []*js.Script{
        js.ServiceWorker(&MySW{}),
    }
}
```

## Runtime Selection

The framework needs to know which `wasm_exec.js` to include. This is configured at boot:

```go
js.SetRuntime(js.RuntimeTinyGo) // or js.RuntimeGo
```
