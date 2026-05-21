package js

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
