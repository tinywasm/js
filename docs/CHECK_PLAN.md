# PLAN — Sync `wasm_exec.js` embeds with Go and TinyGo versions

## Context

`tinywasm/js` embeds two JS runtime files in `js/assets/`:

| File | Source | Current version |
|---|---|---|
| `wasm_exec_go.js` | Go stdlib `misc/wasm/wasm_exec.js` | Go 1.24.x |
| `wasm_exec_tinygo.js` | TinyGo release `targets/wasm_exec.js` | TinyGo 0.40.1 |

These files are the JS↔WASM bridge injected into every shim produced by `PageBootstrap()`,
`ServiceWorker()`, and `WebWorker()`. If the embedded file does not match the compiler
version used to build `client.wasm`, the WASM binary silently fails in the browser at
runtime with no compile-time error.

`tinywasm/tinygo.DefaultVersion` (currently `"0.40.1"`) is the single source of truth for
which TinyGo version the framework targets.

## Solution

Keep the `//go:embed` files as the source of truth. Add a version annotation on line 1 of
each asset, a `go:generate` script that downloads and replaces the assets, and a sync test
that auto-updates assets on disk when versions diverge (requiring a second `gotest` run to
confirm).

## Implementation

### Step 1 — Version annotations

Prepend a version comment to the first line of each asset file:

`js/assets/wasm_exec_tinygo.js` → first line: `// @tinygo-version 0.40.1`  
`js/assets/wasm_exec_go.js` → first line: `// @go-version 1.24.3`  (use the actual Go version from `js/go.mod`)

### Step 2 — `go generate` script

Create `js/scripts/update_wasm_exec.go` (build tag `//go:build ignore`).

The script must:

1. Read `tinygo.DefaultVersion` from `github.com/tinywasm/tinygo` to get the target TinyGo version.
2. Download the TinyGo release archive for that version:
   `https://github.com/tinygo-org/tinygo/releases/download/v{ver}/tinygo{ver}.linux-amd64.tar.gz`
3. Extract `tinygo/targets/wasm_exec.js` from the archive.
4. Prepend `// @tinygo-version {ver}\n` and write to `js/assets/wasm_exec_tinygo.js`.
5. Read the Go version from `js/go.mod` (the `go X.Y` directive).
6. Download the Go source archive: `https://go.dev/dl/go{ver}.src.tar.gz`
7. Extract `go/misc/wasm/wasm_exec.js` from the archive.
8. Prepend `// @go-version {ver}\n` and write to `js/assets/wasm_exec_go.js`.

Add to `js/js.go` (or a dedicated `js/generate.go`):
```go
//go:generate go run scripts/update_wasm_exec.go
```

### Step 3 — Sync test with auto-update

Create `js/tests/wasm_exec_sync_test.go`.

**`TestWasmExecAnnotationPresent`** (no network, always runs):
- Read `wasmExecTinyGoSource` and `wasmExecGoSource` via the exported embed accessors or
  by reading the asset files from disk relative to the test file.
- Assert the first line of each file matches `// @tinygo-version ` and `// @go-version `
  respectively. Fail immediately if missing.

**`TestWasmExecTinyGoInSync`** (requires network, skip with `-short`):
1. Parse `tinygo.DefaultVersion` → `want`.
2. Parse the annotation in `wasm_exec_tinygo.js` → `have`.
3. If `have == want`, compute SHA-256 of the embed content (after stripping the annotation
   line) and compare with the SHA-256 of `targets/wasm_exec.js` inside the TinyGo release
   archive for `want`. If equal → pass.
4. If versions differ **or** hashes differ:
   - Download and extract `targets/wasm_exec.js` from the TinyGo `want` release archive.
   - Write `// @tinygo-version {want}\n{content}` to `../assets/wasm_exec_tinygo.js`
     (path relative to the test file).
   - Call `t.Fatalf("wasm_exec_tinygo.js updated %s → %s. Re-run: gotest ./...", have, want)`.

**`TestWasmExecGoInSync`** (requires network, skip with `-short`):
- Same logic as above but:
  - Source of truth: `go` directive in `js/go.mod` (parse with `go/modfile` or simple string scan).
  - Remote file: `go/misc/wasm/wasm_exec.js` inside `https://go.dev/dl/go{ver}.src.tar.gz`.
  - Asset: `../assets/wasm_exec_go.js`.
  - Annotation: `// @go-version {ver}`.

### Step 4 — README update

Add a section "Updating wasm_exec.js" to `js/README.md`:

```
## Updating wasm_exec.js

When TinyGo or Go releases a new version:

1. Update `DefaultVersion` in `tinywasm/tinygo/tinygo.go` (for TinyGo).
   For Go, update the `go` directive in `js/go.mod`.
2. Run: go generate ./...
3. Run: gotest ./...   (first run updates assets if generate was skipped; second run must pass)
4. Publish: gopush (tinywasm/tinygo first, then tinywasm/js)
```

## Tests summary

| Test | Network | Behaviour on failure |
|---|---|---|
| `TestWasmExecAnnotationPresent` | No | FAIL — annotation missing or malformed |
| `TestWasmExecTinyGoInSync` | Yes (skip `-short`) | Auto-updates asset on disk + FAIL with re-run instruction |
| `TestWasmExecGoInSync` | Yes (skip `-short`) | Auto-updates asset on disk + FAIL with re-run instruction |

## Stages

### Phase 1 — Infrastructure (tinywasm/js)

| # | Task | Done |
|---|---|---|
| 1 | Prepend `// @tinygo-version 0.40.1` to `js/assets/wasm_exec_tinygo.js` and `// @go-version X.Y.Z` to `js/assets/wasm_exec_go.js` (use the Go version from `js/go.mod`) | [ ] |
| 2 | Write `js/scripts/update_wasm_exec.go` (download + extract + annotate both assets) | [ ] |
| 3 | Add `//go:generate go run scripts/update_wasm_exec.go` to `js/js.go` or a dedicated `js/generate.go` | [ ] |
| 4 | Write `js/tests/wasm_exec_sync_test.go` with the three tests described above | [ ] |
| 5 | Run `go generate ./...` inside `tinywasm/js` — verify output matches current assets (baseline) | [ ] |
| 6 | Run `gotest ./...` inside `tinywasm/js` — first and second run must both pass | [ ] |
| 7 | Update `js/README.md` with the update protocol section | [ ] |

### Phase 2 — Upgrade to TinyGo 0.41.1 (end-to-end validation)

| # | Task | Done |
|---|---|---|
| 8 | Install TinyGo 0.41.1 using the `tinywasm/tinygo` CLI: `sudo tinygoinstall -version 0.41.1 -v` | [ ] |
| 9 | Verify installation: `tinygo version` must print `tinygo version 0.41.1 ...` | [ ] |
| 10 | Update `DefaultVersion = "0.41.1"` in `tinywasm/tinygo/tinygo.go` | [ ] |
| 11 | Run `gotest ./...` inside `tinywasm/tinygo` — must pass | [ ] |
| 12 | If stage 11 passes: publish `tinywasm/tinygo` with `gopush` | [ ] |
| 13 | Run `go generate ./...` inside `tinywasm/js` — assets must update to TinyGo 0.41.1 | [ ] |
| 14 | Run `gotest ./...` inside `tinywasm/js` — second run must pass (first run will update + fail, second must pass) | [ ] |
| 15 | If stage 14 passes: publish `tinywasm/js` with `gopush` | [ ] |
