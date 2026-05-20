# PLAN — API mínima `Script` para módulos SSR

## Objetivo

Definir en `tinywasm/js` la estructura `Script` que representa un fragmento de
JavaScript producido por un módulo SSR. Es el equivalente a `*Stylesheet` de
`tinywasm/css`: el tipo de retorno que un módulo expone para que el extractor
de assets decida si el contenido se acopla al bundle global (`script.js`) o se
emite como archivo independiente (p.ej. service worker, web worker).

## Justificación

Hoy `RenderJS() string` impide emitir archivos JS que **deben** estar
separados del bundle. Casos legítimos: service workers (necesitan estar en el
root para definir scope), web workers cargados por URL, archivos de soporte
PWA. Sin un tipo dedicado el usuario debe escribir el archivo a mano, lo cual
contradice la promesa "implementa los métodos y tinywasm/app se encarga".

## Decisiones de diseño (fijas)

- **Cardinalidad:** un módulo puede devolver `[]*Script` (0..N).
- **Ruta de salida:** `Script.Name` es un nombre simple sin separadores. El
  extractor escribe el archivo en la raíz pública (`/public/<Name>`). Esto
  habilita el scope de service workers y simplifica la resolución; un `Name`
  con `/` o `..` es error de validación.
- **Campos:** `Name string` + `Content string`. Nada más. Decisiones de
  minificación recaen en assetmin (por extensión/contenido); el tipo de
  carga (worker/module/classic) lo decide el código consumidor del módulo.
- **Sin auto-inyección:** assetmin no añade `<script src>`. El módulo es
  responsable de registrar el worker o cargar el archivo desde su contenido
  bundleable (`navigator.serviceWorker.register('/sw.js')`).

## API propuesta

```go
package js

// Script representa un fragmento JS producido por un módulo SSR.
// - Name vacío: Content se acopla al bundle global script.js.
// - Name no vacío: Content se escribe como /public/<Name> (archivo independiente).
type Script struct {
    Name    string // nombre simple ("sw.js"); sin separadores ni path traversal.
    Content string
}

// String devuelve el contenido bruto (paridad con tinywasm/css Stylesheet.String()).
func (s *Script) String() string { return s.Content }
```

**Sin método `Valid()`.** La regla "Name sin `/` ni `..`" sólo tiene sentido
en el boundary donde Name se usa como path en el filesystem (assetmin), no
en el tipo. `tinywasm/css.Stylesheet` no tiene `Valid()` por la misma razón:
el paquete es datos, no policía. La validación se ejecuta una sola vez, en
`assetmin` al registrar el Script.

**Sin helpers de construcción.** `Script` se instancia con struct literal:

```go
&js.Script{Content: "console.log('x')"}            // bundle
&js.Script{Name: "sw.js", Content: swSource}       // standalone
```

Helpers tipo `Bundled(content)` / `Standalone(name, content)` serían
boilerplate sin lógica (mismo anti-patrón que `SSRInstance`) y duplicarían la
validación que assetmin ya ejecuta en el boundary de registro.

## Eliminaciones

- Borrar el skeleton actual de `js.go` (`type Js struct{}` + `New()`).

## Tests

Ubicación: `tests/` (subpaquete `js_test` — black-box, importa
`github.com/tinywasm/js`). No mezclar tests con el código del paquete raíz.

| Archivo | Test | Verifica |
|---|---|---|
| `tests/script_test.go` | `TestScript_StringReturnsContent` | `(&js.Script{Content:"x"}).String() == "x"` |
| `tests/script_test.go` | `TestScript_ZeroValue` | `(&js.Script{}).String() == ""` y `Name == ""` |

## Documentación

Crear `README.md` con:
- Propósito (mirror de `tinywasm/css`).
- API (`Script` con `Name`, `Content` y método `String()`).
- Ejemplo de service worker en PWA.
- Restricciones de `Name` (root público, sin separadores).

## Stages

| # | Tarea | Done |
|---|---|---|
| 1 | Reemplazar `js.go` con el tipo `Script` + método `String()` | [ ] |
| 2 | Suite de tests listada arriba — `go test ./...` verde | [ ] |
| 3 | Escribir `README.md` con propósito y ejemplos | [ ] |
| 4 | Publicar tag inicial (`v0.1.0`) para que `go.mod` upstream lo fije | [ ] |
