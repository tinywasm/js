# PLAN — Capa tipada para Service Worker y Web Worker

## Objetivo

Permitir que un módulo SSR declare Service Workers y Web Workers **escribiendo
únicamente Go tipado**. `tinywasm/js` recibe un handler que implementa una
interfaz Go conocida y produce el `*Script` con el JS shim necesario para
puentear los eventos del navegador (FetchEvent, MessageEvent) hacia ese
handler, ejecutándose dentro del mismo WASM que la página.

El tipo `Script{Name, Content}` (v1 ya entregada en
[PLAN_v1_SCRIPT_DELIVERED.md](PLAN_v1_SCRIPT_DELIVERED.md)) queda como escape
hatch de bajo nivel. La API recomendada pasa a ser tipada.

## Justificación

`Script.Content string` re-introduce JS crudo en el código del usuario y rompe
la promesa del framework: "escribe Go, el framework genera lo demás".
`tinywasm/css` resolvió el mismo problema con un DSL tipado; el equivalente
correcto para JS **no** es un DSL JS-en-Go (JS es Turing-completo: reinventar
un compilador no escala y duplica lo que TinyGo ya hace al compilar a WASM),
sino:

> El usuario implementa la lógica como un handler Go tipado.
> `tinywasm/js` genera el JS-shim mínimo que carga WASM en el contexto
> aislado (SW o Worker) y despacha cada evento al método Go correspondiente.

Resultado: tipos en compile-time, autocompletado IDE, refactor seguro, cero
JS escrita por el usuario.

## Decisiones de diseño (fijas)

- **Alcance v2:** Service Worker + Web Worker. Otros casos (inline page JS,
  scripts auxiliares) usan el escape hatch `&Script{Name, Content}` v1.
- **Modelo:** interfaz Go + shim JS generado.
- **WASM en SW/Worker:** el shim carga **el mismo binario** que la página
  (no se compila un WASM separado en v2). Optimización futura: binario
  dedicado más liviano (issue aparte).
- **Eventos tipados propios:** `js.Request`, `js.Response`, `js.Message` —
  el handler nunca toca `syscall/js.Value`.
- **`tinywasm/js` es la única API JS.** Espejo de `tinywasm/css`:
  `RenderJS() []*js.Script` devuelve `Content` **ya final** (string listo
  para escribir a disco/memoria). Sin resolución diferida, sin interfaces extra,
  sin coordinación cross-package en runtime. `assetmin` sólo conoce `js`.
- **`js` posee los embeds de `wasm_exec.js`.** `wasm_exec.js` (Go y TinyGo)
  son JavaScript → pertenecen al paquete JS. Otros paquetes que hoy los
  embeben (notablemente `tinywasm/client`) deberán consumirlos desde aquí.
  Una sola fuente de verdad evita divergencia en actualizaciones de runtime.
- **URL del WASM fija:** constante interna `defaultWasmURL = "/client.wasm"`.
  **No pública** — ningún consumidor la necesita (los shims la incrustan
  literal en el template; `client` sirve `/client.wasm` con string literal
  propio). Si alguna vez cambia, un solo lugar dentro de `js`. Cero
  parámetros al usuario.

## API pública propuesta

### Superficie pública (mínima)

Sólo lo estrictamente necesario para que consumidores externos (módulos
SSR, `tinywasm/app`, `tinywasm/client`) puedan trabajar. Todo lo demás
queda **unexported** dentro del paquete.

```go
package js

// --- Tipo raíz (ya existente) ---
type Script struct {
    Name    string
    Content string
}
func (s *Script) String() string

// --- Runtime activo (escrito por app, leído internamente por los shims) ---
type Runtime int
const (
    RuntimeGo Runtime = iota
    RuntimeTinyGo
)
// SetRuntime: write-once-at-boot. Sin getter público.
func SetRuntime(r Runtime)

// --- Constructores tipados (componen Content final usando el runtime activo) ---
func PageBootstrap() *Script                            // Name="" → bundle /script.js
func ServiceWorker(handler ServiceWorkerHandler) *Script // Name="sw.js"
func WebWorker(name string, handler WebWorkerHandler) *Script

// --- Interfaces handler (implementadas por el usuario) ---
type ServiceWorkerHandler interface { /* ver §"Interfaces de handler" */ }
type WebWorkerHandler interface     { /* idem */ }

// --- Tipos de eventos (usados por las firmas de los handlers) ---
type Request struct { /* ver §"Tipos de eventos"; Headers []fetch.Header */ }
type Message struct { Data []byte }
// La respuesta de OnFetch es *fetch.Response (no hay js.Response propio).
```

### Símbolos **privados** (intencional)

| Símbolo | Por qué privado |
|---|---|
| `defaultWasmURL = "/client.wasm"` | Constante de implementación del shim. Los consumidores nunca la referencian. |
| `wasmExecGo() string`, `wasmExecTinyGo() string` | Sólo los composers internos (`PageBootstrap`/`ServiceWorker`/`WebWorker`) las leen. `client` no las necesita: pasó a ser build-only y ya no compone JS. |
| `activeRuntime() Runtime` | Sólo lo lee el composer interno. Sin getter público porque expone estado mutable global. Si app necesita verificar lo que escribió, su test asserta vía side-effect (compone un Script y verifica qué `wasm_exec` quedó inlineado). |
| `wasmExecGoSignatures []string`, `wasmExecTinyGoSignatures []string` | Constantes internas usadas por los tests del propio paquete `js`. Tests externos declaran inline las firmas que les interesan. |
| `swHandler`, `workerHandlers map[string]...` | Registros internos. |
| Templates JS, helpers de composición de shim | Detalle de implementación. |

### Constructores — detalle de comportamiento

`PageBootstrap()`:
- Devuelve `*Script{Name: "", Content: ...}` — `Name` vacío indica a assetmin
  que el contenido va al bundle `/script.js`.
- Content = `wasm_exec.js` (Go o TinyGo según runtime activo) + bootstrap
  `WebAssembly.instantiateStreaming(fetch("/client.wasm")).then(go.run)`.
- Reemplaza completamente la antigua `client.WasmClient.GetSSRClientInitJS()`.

`ServiceWorker(handler)`:
- Nombre fijo `"sw.js"` (contrato del navegador: un SW por scope; `/sw.js`
  alcanza scope global).
- Compone Content en tiempo de llamada: `wasm_exec.js` correspondiente al
  runtime activo + bootstrap WASM apuntando a `/client.wasm` + listeners
  install/activate/fetch que invocan funciones puente del WASM.
- Sólo un SW handler por aplicación (error al registrar el segundo).

`WebWorker(name, handler)`:
- `name` define el archivo standalone (p.ej. `"parser.worker.js"`).
- Misma composición que SW pero con listener `message`.
- Múltiples workers coexisten con `name` distinto.

### Por qué runtime es estado global y no parámetro

`RenderJS() []*js.Script` no recibe parámetros y los módulos SSR no tienen
forma directa de obtener el modo de compilación. Tres alternativas evaluadas:

1. **Parámetro explícito** (`ServiceWorker(h, runtime)`): obliga al módulo a
   importar `tinywasm/client` para preguntar el runtime activo → mezcla de
   APIs (rechazado por principio 3 del MASTER PLAN).
2. **Auto-detección en runtime WASM**: el shim corre en navegador; en
   tiempo de generación (Go nativo) no hay forma de "detectar" cuál se va a
   compilar después.
3. **Estado global escrito por `app` al boot** (elegido): `app` ya posee el
   modo de compilación. Llama `js.SetRuntime(...)` una sola vez antes de que
   los módulos invoquen `RenderJS()`. El extractor SSR de assetmin corre en
   el mismo proceso → ve el global. Cero parámetros para el usuario.

### Interfaces de handler

```go
import (
    "github.com/tinywasm/context" // stdlib "context" está vetado en wasm
    "github.com/tinywasm/fetch"   // Header y Response canónicos del ecosistema
)

// ServiceWorkerHandler es la lógica del usuario para los eventos del SW.
// Implementaciones parciales son válidas: cualquier método ausente se
// resuelve con el default browser-side (no-op para Install/Activate; pasada
// al network para Fetch).
type ServiceWorkerHandler interface {
    OnInstall(ctx context.Context) error
    OnActivate(ctx context.Context) error
    OnFetch(ctx context.Context, req *Request) (*fetch.Response, error)
}

// WebWorkerHandler procesa mensajes recibidos por postMessage.
type WebWorkerHandler interface {
    OnMessage(ctx context.Context, msg *Message) (*Message, error)
}
```

`OnFetch` devuelve `*fetch.Response` (tipo canónico del ecosistema). El
usuario lo construye con el constructor público de `fetch` (precondición —
ver §"Regla de dependencias"). Para serializar un payload tipado en la
respuesta: `json.Encode(modelo, &buf)` donde `modelo` es un struct generado
por `ormc` (implementa `fmt.Fielder`).

### Regla de dependencias (estricta)

`tinywasm/js` compila a WASM en el navegador. La stdlib está **vetada**
porque infla el binario. Reemplazos obligatorios:

| Stdlib (prohibido) | Reemplazo (obligatorio) |
|---|---|
| `context` | `github.com/tinywasm/context` |
| `fmt`, `errors` | `github.com/tinywasm/fmt` (cubre `Sprintf`, `Errf`, etc.) |
| `strings` | `github.com/tinywasm/fmt` |
| `strconv` | `github.com/tinywasm/fmt` |
| `path` | `github.com/tinywasm/fmt` |
| `encoding/json` | `github.com/tinywasm/json` |
| `time` | `github.com/tinywasm/time` |
| `map[string]string` (headers) | `[]fetch.Header` (`{Key, Value string}`) |

Validar al PR: `grep -rn '"context"\|"fmt"\|"strings"\|"errors"\|"strconv"\|"path"\|"encoding/json"\|"time"' *.go tests/*.go` debe estar vacío (excepto build tags no-wasm si los hubiera).

**Maps prohibidos en tipos públicos.** TinyGo maneja mapas de forma ineficiente
(binario más grande, GC pressure). Headers van como `[]fetch.Header`; payloads
estructurados van como `[]byte` serializado vía `json` hacia structs `ormc`.

**Precondición externa — `tinywasm/fetch` debe exponer un constructor público
de `Response`.** Hoy `fetch.Response.body` es privado (sólo getter `Body()`),
así que el handler no puede construir la respuesta. Añadir a `tinywasm/fetch`:
```go
// NewResponse construye una Response servible desde un Service Worker handler.
func NewResponse(status int, headers []Header, body []byte) *Response
```
Esto es un cambio menor en `tinywasm/fetch` (sin breaking). Registrar como
stage 0 / precondición de este PLAN.

### Tipos de eventos (mínimos)

```go
import "github.com/tinywasm/fetch"

// Request es el FetchEvent entrante interceptado por el SW (inbound).
// NO se reusa fetch.Request (que es un builder de salida con campos privados).
type Request struct {
    URL     string
    Method  string
    Headers []fetch.Header // {Key, Value} — sin maps
    Body    []byte
}

// La respuesta del handler es *fetch.Response (tipo canónico del ecosistema),
// construida con fetch.NewResponse(status, headers, body). No se define un
// js.Response propio.

// Message es el payload de postMessage de un Web Worker. Data viaja
// serializado; el handler lo decodifica a un struct ormc con json.Decode:
//
//   var m MiModelo            // generado por ormc, implementa fmt.Fielder
//   json.Decode(msg.Data, &m)
type Message struct {
    Data []byte
}
```

Patrón de uso tipado (ejemplo `OnFetch`):
```go
func (h *MySW) OnFetch(ctx context.Context, req *js.Request) (*fetch.Response, error) {
    var out MiRespuestaModelo // struct ormc
    // ... llenar out ...
    var buf []byte
    if err := json.Encode(out, &buf); err != nil {
        return nil, err
    }
    return fetch.NewResponse(200, []fetch.Header{{Key: "Content-Type", Value: "application/json"}}, buf), nil
}
```

### Registro interno (lado WASM-página)

Cada llamada a `ServiceWorker(...)` / `WebWorker(...)`:

1. Almacena el handler en un registro global por contexto
   (`swHandler ServiceWorkerHandler`, `workerHandlers map[string]WebWorkerHandler`).
2. Expone funciones globales en el WASM consultables vía `syscall/js`:
   - `__tinywasm_sw_fetch`, `__tinywasm_sw_install`, `__tinywasm_sw_activate`
   - `__tinywasm_worker_message_<name>`
3. Devuelve un `*Script` con el shim que invoca esas funciones.

El bootstrap del WASM detecta su contexto (`self.constructor.name ===
"ServiceWorkerGlobalScope"` etc.) para evitar inicializar código de DOM en
contextos sin window/document.

## Composición del shim (auto-contenida en `js`)

El SW corre en `ServiceWorkerGlobalScope` y **no** puede usar el bundle
`/script.js` de la página (contiene código DOM). El shim de SW es un archivo
standalone `/sw.js` con el runtime Go inlineado.

`js.ServiceWorker(handler, runtime)` compone el string **en tiempo de
llamada** usando los embeds que `js` ya posee (`assets/wasm_exec_go.js`,
`assets/wasm_exec_tinygo.js`). El `*Script` devuelto tiene `Content` final.
assetmin lo escribe tal cual — sin interfaces extra, sin coordinación.

`sw.js` resultante (ilustrativo):

```js
// Generated by tinywasm/js — do not edit.
<contenido literal de wasm_exec_go.js o wasm_exec_tinygo.js>
const go = new Go();
WebAssembly.instantiateStreaming(fetch('/client.wasm'), go.importObject)
  .then(r => { go.run(r.instance); });

self.addEventListener('install',  e => e.waitUntil(self.__tinywasm_sw_install()));
self.addEventListener('activate', e => e.waitUntil(self.__tinywasm_sw_activate()));
self.addEventListener('fetch',    e => e.respondWith(self.__tinywasm_sw_fetch(e.request)));
```

Notas:
- `wasm_exec.js` se inlinea (no `importScripts`) porque assetmin no lo
  publica como archivo separado en raíz (va bundled en `/script.js`).
- `/client.wasm` es la ruta fija de v2 (`js.DefaultWasmURL`).
- No hay JS legacy ni `importScripts` externos dentro del SW en v2.
- Web Worker reutiliza el mismo patrón (string final al construir).
- Si el handler necesita refrescarse al cambiar `wasm_exec.js` (hot reload
  durante desarrollo), el llamante reconstruye el `*Script` — `js` no
  mantiene caché.

## Eliminaciones

- Ninguna. La API v1 (`Script` struct + `String()`) permanece publicada como
  escape hatch.

## Restricciones / problemas conocidos

- **SW y Worker no comparten estado con la página.** Cada contexto carga su
  propia instancia del WASM (mismo binario, distinta heap). Documentar
  claramente que el handler corre aislado.
- **Inicialización condicional del runtime Go.** Código en `init()` que
  toque DOM debe estar guardado por `//go:build wasm` y comprobación de
  contexto. Añadir a `tinywasm/wasm` (skill) la regla correspondiente.
- **Tamaño del binario.** Cargar el WASM completo en el SW puede ser
  costoso para PWA offline-first. Se acepta en v2; v3 optimizará con WASM
  dedicado.
- **Una sola implementación de `ServiceWorkerHandler` por app.** Por
  contrato del navegador (un SW por scope). Llamar a `ServiceWorker(...)`
  dos veces en la misma app es error de registro detectado por assetmin.

## Tests

Ubicación: `tests/` (subpaquete `js_test` — black-box).

| Archivo | Test | Verifica |
|---|---|---|
| `tests/service_worker_test.go` | `TestServiceWorker_GeneratesShim` | `ServiceWorker(h)` devuelve `*Script` con `Name="sw.js"` (fijo) y `Content` no vacío |
| `tests/service_worker_test.go` | `TestServiceWorker_ShimContainsHooks` | El shim referencia `__tinywasm_sw_fetch/install/activate` |
| `tests/service_worker_test.go` | `TestServiceWorker_DuplicateRegistration` | Dos llamadas → error explícito |
| `tests/web_worker_test.go` | `TestWebWorker_GeneratesShim` | `WebWorker("p.js", h)` devuelve `*Script` correcto |
| `tests/web_worker_test.go` | `TestWebWorker_MultipleWorkers` | Dos workers con names distintos coexisten |
| `tests/types_test.go` | `TestRequest_Roundtrip` | Marshal/unmarshal desde un js.Value de prueba |
| `tests/types_test.go` | `TestResponse_Roundtrip` | Idem para Response |
| `tests/handler_dispatch_test.go` | `TestDispatch_FetchInvokesHandler` | (build wasm) un fetch simulado llama a `OnFetch` y devuelve la Response |

### Ejecución de tests

- **Stdlib (lógica pura: registro, generación de shim, dispatch sintético):**
  `gotest ./...` — corre nativo en Go estándar.
- **WASM en navegador (handlers reales bajo `//go:build wasm`, ciclo SW
  install→activate→fetch, postMessage real a un Web Worker):**
  `gotest -wasm ./...` — la skill [[testing]] (`gotest`) compila a WASM,
  arranca un navegador headless y reporta resultados. Es el único modo que
  ejerce `__tinywasm_sw_*` con `syscall/js` reales y el bootstrap por
  contexto (`ServiceWorkerGlobalScope` / `DedicatedWorkerGlobalScope`).
- Publicación del paquete tras verde dual: `gopush` (skill [[testing]]).

Convención: todo archivo con tests que toquen `syscall/js` debe llevar
`//go:build wasm` en la cabecera para que el modo stdlib los excluya.

## Documentación

- Actualizar `README.md` con:
  - Concepto: "escribes el handler en Go, el framework genera el shim".
  - Ejemplo Service Worker completo (PWA básica).
  - Ejemplo Web Worker (cálculo pesado).
  - Sección "escape hatch": cuándo usar `&Script{...}` directo en vez de los
    constructores tipados.
- Añadir `docs/ARCHITECTURE.md` con diagrama del flujo:
  página WASM ↔ shim JS ↔ contexto aislado (SW/Worker) WASM.
- Nota en `docs/PLAN_v1_SCRIPT_DELIVERED.md`: "API base — la capa tipada se
  documenta en PLAN.md (v2)".

## Compatibilidad con consumidores

`RenderJS() []*js.Script` no cambia. Los consumidores siguen devolviendo
slices de `*Script`; lo único que cambia es **cómo** los construyen:

```go
// Bundle inline en /script.js (sin cambios respecto a v1) — único canal
// soportado para JS legacy/arbitrario (analytics, polyfills, init libs).
&js.Script{Content: "console.log('x')"}

// Archivo standalone crudo (escape hatch v1) — el usuario gestiona su shim.
&js.Script{Name: "raw.js", Content: customJS}

// Nuevo: Service Worker tipado (nombre "sw.js" fijo). Devuelve *Script con
// Content final, listo para assetmin. El runtime se lee de js.ActiveRuntime()
// (configurado por tinywasm/app al boot).
js.ServiceWorker(&MyAppSW{})

// Nuevo: Web Worker tipado.
js.WebWorker("parser.worker.js", &ParserWorker{})
```

Impacto en consumidores (informativo — fuera del alcance de este PLAN):
- **assetmin:** ninguno funcional. `RenderJS() []*js.Script` con `Content`
  final — exactamente como `tinywasm/css`.
- **client:** pasa a ser **build-only**. Borrar toda la composición JS
  (`Javascript` struct, `GetSSRClientInitJS`, `SetMode`, `SetWasmFilename`,
  embeds locales de `wasm_exec_*.js`). Sólo conserva: compilación WASM,
  detección Go/TinyGo (para informar a `app`), watcher, `RegisterRoutes`
  para `/client.wasm`.
- **app (orquestador):** (1) invoca `js.SetRuntime(...)` al detectar el
  modo de compilación; (2) registra un `js.PageBootstrap()` Script con
  assetmin directamente (reemplazo del callback `GetSSRClientInitJS` actual).
- **dom, site, components, layout:** sin cambios.

## Stages

| # | Tarea | Done |
|---|---|---|
| 0 | **Precondición externa:** publicar `tinywasm/fetch` con `func NewResponse(status int, headers []Header, body []byte) *Response` | [ ] |
| 1 | Definir tipos `Request` (`Headers []fetch.Header`), `Message` (`Data []byte`). La respuesta de OnFetch es `*fetch.Response` | [ ] |
| 2 | Definir interfaces `ServiceWorkerHandler`, `WebWorkerHandler` | [ ] |
| 3 | Implementar registro global de handlers con detección de duplicados | [ ] |
| 4 | Implementar funciones puente `__tinywasm_sw_*` y `__tinywasm_worker_*` (build `wasm`) | [ ] |
| 5a | Añadir `require` de `github.com/tinywasm/fmt`, `tinywasm/context`, `tinywasm/fetch`, `tinywasm/json` al `js/go.mod` | [ ] |
| 5b | Migrar embeds `wasm_exec_go.js` / `wasm_exec_tinygo.js` desde `client/assets/` a `js/assets/` con getters **privados** `wasmExecGo()` / `wasmExecTinyGo()` | [ ] |
| 5c | Exponer mínima superficie pública: `Runtime`/`RuntimeGo`/`RuntimeTinyGo`, `SetRuntime`. Resto privado. | [ ] |
| 6a | Constructor `PageBootstrap()` que compone `wasm_exec` + `WebAssembly.instantiateStreaming(fetch("/client.wasm"))` y devuelve `*Script{Name:""}` | [ ] |
| 6b | Constructores `ServiceWorker(handler)` y `WebWorker(name, handler)` que componen sus shims usando el runtime activo y devuelven `*Script` con `Content` final | [ ] |
| 7 | Detección de contexto (page/SW/worker) en el bootstrap del runtime | [ ] |
| 8 | Suite de tests en `tests/` — `gotest ./...` y `gotest -wasm ./...` verdes (skill [[testing]]) | [ ] |
| 9 | Actualizar `README.md` + crear `docs/ARCHITECTURE.md` | [ ] |
| 10 | Publicar tag `v0.2.0` | [ ] |
| 11 | Validación E2E: PWA mínima en `goflare-demo` con SW que cachea estáticos | [ ] |
