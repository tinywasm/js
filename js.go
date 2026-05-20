package js

// Script representa un fragmento JS producido por un módulo SSR.
// - Name vacío: Content se acopla al bundle global script.js.
// - Name no vacío: Content se escribe como /public/<Name> (archivo independiente).
type Script struct {
	Name    string // nombre simple ("sw.js"); sin separadores ni path traversal.
	Content string
}

// String devuelve el contenido bruto (paridad con tinywasm/css Stylesheet.String()).
func (s *Script) String() string {
	return s.Content
}
