package js

import (
	_ "embed"
	"github.com/tinywasm/context"
	"github.com/tinywasm/fetch"
	. "github.com/tinywasm/fmt"
)

// Script represents a JS fragment produced by an SSR module.
// - Empty Name: Content is bundled into the global script.js.
// - Non-empty Name: Content is written as /public/<Name> (standalone file).
type Script struct {
	Name    string // Simple filename (e.g., "sw.js"); no separators or path traversals.
	Content string
}

// String returns the raw content (for parity with tinywasm/css Stylesheet.String()).
func (s *Script) String() string {
	return s.Content
}

// validate ensures Name is a simple filename.
func (s *Script) validate() {
	if s.Name == "" {
		return
	}
	if Contains(s.Name, "/") || Contains(s.Name, "\\") || Contains(s.Name, "..") {
		panic("js: Script.Name must be a simple filename, got " + s.Name)
	}
}

// Request is the FetchEvent incoming intercepted by the SW (inbound).
type Request struct {
	URL     string
	Method  string
	Headers []fetch.Header
	Body    []byte
}

// Message is the payload of postMessage of a Web Worker.
type Message struct {
	Data []byte
}

// ServiceWorkerHandler is the logic for SW events.
type ServiceWorkerHandler interface {
	OnInstall(ctx context.Context) error
	OnActivate(ctx context.Context) error
	OnFetch(ctx context.Context, req *Request) (*fetch.Response, error)
}

// WebWorkerHandler processes messages received by postMessage.
type WebWorkerHandler interface {
	OnMessage(ctx context.Context, msg *Message) (*Message, error)
}

// Runtime represents the Go compiler used.
type Runtime int

const (
	RuntimeGo Runtime = iota
	RuntimeTinyGo
)

var activeRuntime = RuntimeGo

// SetRuntime configures the runtime once at boot.
func SetRuntime(r Runtime) {
	activeRuntime = r
}

var (
	swHandler      ServiceWorkerHandler
	workerHandlers = make(map[string]WebWorkerHandler)
)

// --- Embeds ---

//go:embed assets/wasm_exec_go.js
var wasmExecGoSource string

//go:embed assets/wasm_exec_tinygo.js
var wasmExecTinyGoSource string

func wasmExecGo() string     { return wasmExecGoSource }
func wasmExecTinyGo() string { return wasmExecTinyGoSource }

const defaultWasmURL = "/client.wasm"

// PageBootstrap returns the entrypoint Script for the main page.
func PageBootstrap() *Script {
	s := &Script{Name: ""}
	s.validate()

	runtimeJS := wasmExecGo()
	if activeRuntime == RuntimeTinyGo {
		runtimeJS = wasmExecTinyGo()
	}

	content := runtimeJS + `
if (self.constructor.name === "Window") {
const go = new Go();
if (WebAssembly.instantiateStreaming) {
	WebAssembly.instantiateStreaming(fetch("` + defaultWasmURL + `"), go.importObject).then((result) => {
		go.run(result.instance);
	});
} else {
	fetch("` + defaultWasmURL + `").then(response =>
		response.arrayBuffer()
	).then(bytes =>
		WebAssembly.instantiate(bytes, go.importObject)
	).then(result => {
		go.run(result.instance);
	});
}
}
`
	return &Script{Name: "", Content: content}
}

// ServiceWorker returns the standalone Script for the service worker.
func ServiceWorker(handler ServiceWorkerHandler) *Script {
	s := &Script{Name: "sw.js"}
	s.validate()

	swHandler = handler

	runtimeJS := wasmExecGo()
	if activeRuntime == RuntimeTinyGo {
		runtimeJS = wasmExecTinyGo()
	}

	content := runtimeJS + `
const go = new Go();
WebAssembly.instantiateStreaming(fetch("` + defaultWasmURL + `"), go.importObject).then((result) => {
	go.run(result.instance);
});

self.addEventListener('install',  e => {
    if (self.__tinywasm_sw_install) {
        e.waitUntil(self.__tinywasm_sw_install());
    }
});
self.addEventListener('activate', e => {
    if (self.__tinywasm_sw_activate) {
        e.waitUntil(self.__tinywasm_sw_activate());
    }
});
self.addEventListener('fetch',    e => {
    if (self.__tinywasm_sw_fetch) {
        e.respondWith(self.__tinywasm_sw_fetch(e.request));
    }
});
`
	return &Script{Name: "sw.js", Content: content}
}

// WebWorker returns a standalone Script for a web worker.
func WebWorker(name string, handler WebWorkerHandler) *Script {
	s := &Script{Name: name}
	s.validate()

	workerHandlers[name] = handler

	runtimeJS := wasmExecGo()
	if activeRuntime == RuntimeTinyGo {
		runtimeJS = wasmExecTinyGo()
	}

	content := runtimeJS + `
const go = new Go();
WebAssembly.instantiateStreaming(fetch("` + defaultWasmURL + `"), go.importObject).then((result) => {
	go.run(result.instance);
});

self.addEventListener('message', e => {
    if (self.__tinywasm_worker_message) {
        self.__tinywasm_worker_message("` + name + `", e.data);
    }
});
`
	return &Script{Name: name, Content: content}
}
