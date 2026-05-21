# tinywasm/js

Minimal API for TinyWASM SSR modules that generate JavaScript fragments. It's the equivalent of `tinywasm/css` but for scripting assets.

## Purpose

The core `Script` type represents JS code emitted by a module. By returning a `Script`, we allow the asset extractor (`assetmin`) to decide if the content:
1. Is bundled into the global `script.js` file.
2. Is written as a standalone file in the public root (ideal for service workers, web workers, etc.).

## API

The package exposes the `Script` struct with two fields:

```go
type Script struct {
    Name    string // Simple filename for a standalone file.
    Content string // Raw JavaScript code.
}

func (s *Script) String() string { return s.Content }
```

### Rules for `Name`

- **Empty:** The content of `Content` is automatically bundled into the global JS bundle.
- **Non-empty:** The content is saved as a file in the public root (`/public/<Name>`). It must be a simple filename, without path separators (`/`, `\`) or path traversals (`..`).

## Usage Example (Service Worker in a PWA)

In this scenario, a PWA module registers the service worker through a script injected into the global bundle, while the service worker code itself is returned as a separate file to define the correct scope (the root of the site).

```go
package pwa

import "github.com/tinywasm/js"

type Module struct{}

// RenderJS exposes the JavaScript fragments.
func (m Module) RenderJS() []*js.Script {
	return []*js.Script{
		{
			// Bundleable: injected into the global entrypoint.
			Content: `if ('serviceWorker' in navigator) {
                navigator.serviceWorker.register('/sw.js');
            }`,
		},
		{
			// Standalone file (non-empty Name).
			Name: "sw.js",
			Content: `self.addEventListener('fetch', function(event) {
                console.log('SW fetch:', event.request.url);
            });`,
		},
	}
}
```
