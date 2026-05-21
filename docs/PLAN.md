# PLAN — Fix sync tests timeout + publish `tinywasm/js`

## Módulo

`github.com/tinywasm/js` — ubicado en `tinywasm/js/`.

## Contexto (ya implementado — no repetir)

El paquete está completamente implementado:

- `Script`, `Request`, `Message`, `Runtime`, `SetRuntime` — en `js.go`.
- `PageBootstrap()`, `ServiceWorker()`, `WebWorker()` — en `js.go`.
- Embeds `assets/wasm_exec_go.js` (anotación `// @go-version 1.25.2`) y
  `assets/wasm_exec_tinygo.js` (anotación `// @tinygo-version 0.41.1`).
- `scripts/update_wasm_exec.go` + `//go:generate` en `js.go`.
- Bridge WASM en `js_wasm.go`.
- Tests de API en `tests/wasm_exec_test.go` y `tests/script_test.go`.
- Tests de sincronización en `tests/wasm_exec_sync_test.go` (aquí está el bug).
- `README.md` completo.

## Problema

`gotest ./...` en `tinywasm/js/tests/wasm_exec_sync_test.go` termina con:

```
panic: test timed out after 30s
running tests: TestWasmExecTinyGoInSync (30s)
```

**Causa raíz:** `TestWasmExecTinyGoInSync` y `TestWasmExecGoInSync` descargan
el tarball completo de la release (TinyGo: ~170 MB; Go source: ~30 MB) de forma
**incondicional**, incluso cuando la anotación de versión en el asset ya coincide
con `tinygo.DefaultVersion` / versión en `go.mod`. La descarga ocurre antes del
check, y 30 s no es suficiente.

## Fix requerido

**Archivo:** `tests/wasm_exec_sync_test.go`

**Regla:** si la anotación de versión embebida ya coincide con la versión
objetivo (`have == want`), el test **pasa sin descarga**. Solo descargar cuando
hay divergencia de versión, para actualizar el asset en disco (comportamiento
actual de auto-update).

El hash-check puede eliminarse: la integridad del contenido la garantiza el
script `go generate`; la sync-test solo guarda contra "cambiaste DefaultVersion
sin correr generate".

### `TestWasmExecTinyGoInSync` — lógica corregida

```go
func TestWasmExecTinyGoInSync(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping in short mode")
    }

    want := tinygo.DefaultVersion
    data, err := os.ReadFile("../assets/wasm_exec_tinygo.js")
    if err != nil {
        t.Fatalf("failed to read asset: %v", err)
    }

    lines := strings.SplitN(string(data), "\n", 2)
    have := ""
    if strings.HasPrefix(lines[0], "// @tinygo-version ") {
        have = strings.TrimSpace(lines[0][len("// @tinygo-version "):])
    }

    // Si la versión ya coincide → el asset está al día. No descargar.
    if have == want {
        return
    }

    // Versiones divergen → descargar y actualizar el asset en disco.
    url := fmt.Sprintf(
        "https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.linux-amd64.tar.gz",
        want, want,
    )
    remoteData, err := downloadAndExtract(url, "tinygo/targets/wasm_exec.js")
    if err != nil {
        t.Fatalf("failed to download remote version: %v", err)
    }
    newContent := fmt.Sprintf("// @tinygo-version %s\n%s", want, string(remoteData))
    if err := os.WriteFile("../assets/wasm_exec_tinygo.js", []byte(newContent), 0644); err != nil {
        t.Fatalf("failed to update asset: %v", err)
    }
    t.Fatalf("wasm_exec_tinygo.js updated %s -> %s. Re-run: gotest ./...", have, want)
}
```

### `TestWasmExecGoInSync` — misma lógica

```go
func TestWasmExecGoInSync(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping in short mode")
    }

    want, err := getGoVersion()
    if err != nil {
        t.Fatalf("failed to get go version: %v", err)
    }

    data, err := os.ReadFile("../assets/wasm_exec_go.js")
    if err != nil {
        t.Fatalf("failed to read asset: %v", err)
    }

    lines := strings.SplitN(string(data), "\n", 2)
    have := ""
    if strings.HasPrefix(lines[0], "// @go-version ") {
        have = strings.TrimSpace(lines[0][len("// @go-version "):])
    }

    // Si la versión ya coincide → el asset está al día. No descargar.
    if have == want {
        return
    }

    // Versiones divergen → descargar y actualizar.
    url := fmt.Sprintf("https://go.dev/dl/go%s.src.tar.gz", want)
    remoteData, err := downloadAndExtract(url, "go/lib/wasm/wasm_exec.js")
    if err != nil {
        t.Fatalf("failed to download remote version: %v", err)
    }
    newContent := fmt.Sprintf("// @go-version %s\n%s", want, string(remoteData))
    if err := os.WriteFile("../assets/wasm_exec_go.js", []byte(newContent), 0644); err != nil {
        t.Fatalf("failed to update asset: %v", err)
    }
    t.Fatalf("wasm_exec_go.js updated %s -> %s. Re-run: gotest ./...", have, want)
}
```

## Verificación

```bash
gotest ./...
```

Deben pasar los 5 tests sin timeout:
- `TestWasmExecAnnotationPresent` — verifica anotaciones (sin red)
- `TestWasmExecTinyGoInSync` — early-return porque 0.41.1 == 0.41.1
- `TestWasmExecGoInSync` — early-return porque 1.25.2 == 1.25.2
- `TestPageBootstrap_IsBundleScript`, `TestPageBootstrap_ReferencesClientWasm`, etc.

## Publicar

Una vez `gotest ./...` verde:

```bash
gopush
```

## Stages

| # | Tarea | Done |
|---|---|---|
| 1 | Aplicar el fix en `tests/wasm_exec_sync_test.go`: early-return cuando `have == want` en ambas funciones | [ ] |
| 2 | `gotest ./...` verde — todos los tests pasan sin timeout ni red | [ ] |
| 3 | `gopush` — publicar `tinywasm/js` | [ ] |
