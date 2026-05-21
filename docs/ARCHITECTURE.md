# Architecture — tinywasm/js

## Flujo general

```mermaid
flowchart TD
    APP["tinywasm/app (boot)\njs.SetRuntime(RuntimeGo | RuntimeTinyGo)"]

    APP --> PB["js.PageBootstrap()"]
    APP --> SW["js.ServiceWorker(h)"]
    APP --> WW["js.WebWorker(name, h)"]

    RT[("activeRuntime\n(global)")]
    PB -.lee.- RT
    SW -.lee.- RT
    WW -.lee.- RT

    PB --> S1["*Script{Name: ''}\n→ bundle /script.js"]
    SW --> S2["*Script{Name: 'sw.js'}\n→ standalone /sw.js"]
    WW --> S3["*Script{Name: name}\n→ standalone /name"]

    S1 --> AM["assetmin\nContent final — escribe a disco"]
    S2 --> AM
    S3 --> AM
```

## Contextos de ejecución WASM

El mismo binario `client.wasm` se carga en tres contextos distintos del navegador. Cada contexto es un scope JS aislado — no pueden compartir la instancia WASM entre sí.

```mermaid
flowchart LR
    subgraph WIN["Browser Window (DOM)"]
        direction TB
        S1["/script.js\nwasm_exec inline + bootstrap\ndetector: 'Window'"]
        W1["client.wasm\ninstancia A"]
        S1 --> W1
        W1 --> D1["init() DOM\nswHandler\nworkerHandlers"]
    end

    subgraph SW["ServiceWorkerGlobalScope"]
        direction TB
        S2["/sw.js\nwasm_exec inline + bootstrap\n+ SW listeners"]
        W2["client.wasm\ninstancia B"]
        S2 --> W2
        W2 --> D2["__tinywasm_sw_install\n__tinywasm_sw_activate\n__tinywasm_sw_fetch"]
    end

    subgraph WW["DedicatedWorkerGlobalScope"]
        direction TB
        S3["/parser.worker.js\nwasm_exec inline + bootstrap\n+ message listener"]
        W3["client.wasm\ninstancia C"]
        S3 --> W3
        W3 --> D3["__tinywasm_worker_message(name, data)\n→ workerHandlers[name]"]
    end
```

El shim detecta el contexto vía `self.constructor.name` para evitar ejecutar código DOM en SW/Worker.

## Separación de responsabilidades

| Paquete | Responsabilidad |
|---|---|
| `tinywasm/js` | Composición JS (shims, embeds, constructores tipados). **Única fuente de wasm_exec.js** |
| `tinywasm/client` | Compilación WASM (Go/TinyGo), serving `/client.wasm`. **Sin JS** |
| `tinywasm/app` | Orquestación: llama `js.SetRuntime`, registra `js.PageBootstrap()` con assetmin |
| `assetmin` | Bundling: recibe `[]*js.Script` con `Content` final, escribe a disco |

## Registro de handlers (lado WASM)

```mermaid
flowchart LR
    subgraph HOST["Host (Go nativo — SSR)"]
        SW["js.ServiceWorker(handler)"]
        WW["js.WebWorker(name, handler)"]
    end

    subgraph WASM["WASM (navegador — js_wasm.go init)"]
        SWG[("swHandler\n(global)")]
        WWG[("workerHandlers[name]\n(global)")]

        SWG --> FI["__tinywasm_sw_install\n→ OnInstall"]
        SWG --> FA["__tinywasm_sw_activate\n→ OnActivate"]
        SWG --> FF["__tinywasm_sw_fetch\n→ OnFetch → *fetch.Response"]

        WWG --> FM["__tinywasm_worker_message(name, data)\n→ OnMessage"]
    end

    SW -- "guarda handler" --> SWG
    WW -- "guarda handler" --> WWG
```

## Decisiones clave

- **`activeRuntime` es estado global write-once**: `app` lo escribe una vez al boot antes de que los módulos llamen `RenderJS()`. El extractor SSR de assetmin corre en el mismo proceso → ve el global. Sin parámetros para el usuario.
- **`Content` es string final**: `assetmin` lo escribe tal cual. Sin interfaces extra, sin resolución diferida. Idéntico al modelo de `tinywasm/css`.
- **Sin `wasm_exec.js` en disco**: el archivo se inlinea en cada shim. No hay ruta `/wasm_exec.js` pública.
- **`client/assets/` eliminado**: `js/assets/` es la única fuente de verdad. Un solo lugar a actualizar cuando Go o TinyGo publican nuevas versiones del runtime.
