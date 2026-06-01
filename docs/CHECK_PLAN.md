# PLAN: tinywasm/js — use strict helpers (Tier 2 desde assetmin)

## Repositorio
`github.com/tinywasm/js` — path local: `tinywasm/js/`

## Contexto
Refactor de responsabilidades: assetmin delega lógica JS-específica a `tinywasm/js`
(ver assetmin PLAN Cambio 8).

---

## Cambio 1: Mover `use strict` desde assetmin

`assetmin/use_strict.go` (`stripLeadingUseStrict`) y el prefijo global `'use strict';`
(en `assetmin/events.go: startCodeJS`) son lógica JS → pertenecen a `tinywasm/js`.

Crear `js/use_strict.go`:
```go
package js

// UseStrictPrefix es la directiva global que se antepone al bundle JS.
const UseStrictPrefix = "'use strict';"

// StripLeadingUseStrict elimina una directiva "use strict" al inicio de un archivo JS
// (incluso precedida por comentarios/espacios), para evitar duplicarla al concatenar bundles.
func StripLeadingUseStrict(b []byte) []byte { /* lógica actual de stripLeadingUseStrict */ }
```

En assetmin:
- `events.go`: `file.Content = stripLeadingUseStrict(...)` → `js.StripLeadingUseStrict(...)`
- `events.go: startCodeJS()`: `out = "'use strict';"` → `out = js.UseStrictPrefix`
- `inspect.go`: idem
- Eliminar `assetmin/use_strict.go`. assetmin ya importa `tinywasm/js`.

---

## Tests
Mover los casos de `assetmin` que cubrían `stripLeadingUseStrict` a `js/use_strict_test.go`
(`package js_test`): directiva con comillas dobles/simples, precedida de comentarios, ausencia
de directiva, etc.

## Verificación
```bash
cd tinywasm/js
go build ./...
gotest
gopush
```

Ver `tinywasm/docs/MASTER_PLAN.md` para el orden global.
