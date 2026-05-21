//go:build wasm

package js

import (
	"syscall/js"

	"github.com/tinywasm/context"
	"github.com/tinywasm/fetch"
)

func init() {
	js.Global().Set("__tinywasm_sw_install", js.FuncOf(swInstall))
	js.Global().Set("__tinywasm_sw_activate", js.FuncOf(swActivate))
	js.Global().Set("__tinywasm_sw_fetch", js.FuncOf(swFetch))
	js.Global().Set("__tinywasm_worker_message", js.FuncOf(workerMessage))
}

func swInstall(this js.Value, args []js.Value) any {
	if swHandler == nil {
		return nil
	}
	// Simplified: in a real implementation we would handle the promise
	err := swHandler.OnInstall(context.Background())
	if err != nil {
		return err.Error()
	}
	return nil
}

func swActivate(this js.Value, args []js.Value) any {
	if swHandler == nil {
		return nil
	}
	err := swHandler.OnActivate(context.Background())
	if err != nil {
		return err.Error()
	}
	return nil
}

func swFetch(this js.Value, args []js.Value) any {
	if swHandler == nil {
		// Default behavior: fetch from network
		return js.Global().Call("fetch", args[0])
	}

	reqJS := args[0]
	req := &Request{
		URL:    reqJS.Get("url").String(),
		Method: reqJS.Get("method").String(),
	}
	// TODO: headers and body

	resp, err := swHandler.OnFetch(context.Background(), req)
	if err != nil {
		return js.Global().Call("fetch", args[0])
	}
	if resp == nil {
		return js.Global().Call("fetch", args[0])
	}

	// Convert fetch.Response to JS Response
	// This is a complex part that would involve NewResponse in JS
	return nil // Placeholder
}

func workerMessage(this js.Value, args []js.Value) any {
	name := args[0].String()
	data := args[1] // js.Value

	handler, ok := workerHandlers[name]
	if !ok {
		return nil
	}

	msg := &Message{
		// TODO: convert data to []byte
	}

	res, err := handler.OnMessage(context.Background(), msg)
	if err != nil {
		return nil
	}
	if res != nil {
		// self.postMessage(res.Data)
	}

	return nil
}
