package js_test

import (
	"bytes"
	"testing"

	"github.com/tinywasm/js"
)

func TestStripLeadingUseStrict(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single quotes",
			input:    "'use strict';\nconsole.log(1);",
			expected: "\nconsole.log(1);",
		},
		{
			name:     "Double quotes",
			input:    "\"use strict\";\nconsole.log(1);",
			expected: "\nconsole.log(1);",
		},
		{
			name:     "Without semicolon",
			input:    "'use strict'\nconsole.log(1);",
			expected: "\nconsole.log(1);",
		},
		{
			name:     "With leading whitespace",
			input:    "  \n\t 'use strict';\nconsole.log(1);",
			expected: "\nconsole.log(1);",
		},
		{
			name:     "With leading line comment",
			input:    "// comment\n'use strict';\nconsole.log(1);",
			expected: "\nconsole.log(1);",
		},
		{
			name:     "With leading block comment",
			input:    "/* comment */\n'use strict';\nconsole.log(1);",
			expected: "\nconsole.log(1);",
		},
		{
			name:     "Multiple comments and whitespace",
			input:    "  // line\n/* block */  \n 'use strict';\nconsole.log(1);",
			expected: "\nconsole.log(1);",
		},
		{
			name:     "No directive",
			input:    "console.log(1);",
			expected: "console.log(1);",
		},
		{
			name:     "Directive not at start",
			input:    "console.log(1);\n'use strict';",
			expected: "console.log(1);\n'use strict';",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "Short input",
			input:    "'use strict'",
			expected: "",
		},
		{
			name:     "Mismatched quotes",
			input:    "'use strict\";",
			expected: "'use strict\";",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := js.StripLeadingUseStrict([]byte(tt.input))
			if !bytes.Equal(got, []byte(tt.expected)) {
				t.Errorf("StripLeadingUseStrict() = %q, want %q", string(got), tt.expected)
			}
		})
	}
}
