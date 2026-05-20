# tinywasm/js

API mínima para módulos SSR de TinyWASM que generan fragmentos JavaScript. Es el equivalente a `tinywasm/css` pero para assets de scripting.

## Propósito

El tipo principal `Script` representa código JS emitido por un módulo. Al definir un `Script`, permitimos que el extractor de assets (`assetmin`) decida si el contenido:
1. Se integra al bundle global (`script.js`).
2. Se escribe como archivo independiente en el raíz público (ideal para service workers, web workers, etc.).

## API

El paquete expone la estructura `Script` con dos campos:

```go
type Script struct {
    Name    string // Nombre simple para archivo independiente.
    Content string // Código JavaScript en crudo.
}

func (s *Script) String() string { return s.Content }
```

### Reglas para `Name`

- **Vacío:** El contenido de `Content` se acopla automáticamente al bundle global.
- **No vacío:** El contenido se guarda como un archivo en la raíz pública (`/public/<Name>`). Debe ser un nombre de archivo simple, sin separadores de ruta (`/`, `\`) ni path traversal (`..`).

## Ejemplo de Uso (Service Worker en PWA)

En este escenario, el módulo de PWA registra el service worker a través de un script que se inyecta en el bundle global, mientras que el código del service worker en sí se devuelve como un archivo separado para definir el scope correcto (raíz del sitio).

```go
package pwa

import "github.com/tinywasm/js"

type Module struct{}

// RenderJS expone los fragmentos de JavaScript.
func (m Module) RenderJS() []*js.Script {
	return []*js.Script{
		{
			// Bundleable: inyectado en el entrypoint global.
			Content: `if ('serviceWorker' in navigator) {
                navigator.serviceWorker.register('/sw.js');
            }`,
		},
		{
			// Archivo independiente (Name no vacío).
			Name: "sw.js",
			Content: `self.addEventListener('fetch', function(event) {
                console.log('SW fetch:', event.request.url);
            });`,
		},
	}
}
```
