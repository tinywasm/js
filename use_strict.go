package js

// UseStrictPrefix is the global directive that is prepended to the JS bundle.
const UseStrictPrefix = "'use strict';"

// StripLeadingUseStrict removes a "use strict" directive at the beginning of a JS file
// (even preceded by comments/spaces), to avoid duplicating it when concatenating bundles.
func StripLeadingUseStrict(b []byte) []byte {
	i := 0
	for i < len(b) {
		c := b[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}
		if i+1 < len(b) && c == '/' {
			if b[i+1] == '/' {
				i += 2
				for i < len(b) && b[i] != '\n' {
					i++
				}
				continue
			}
			if b[i+1] == '*' {
				i += 2
				found := false
				for i+1 < len(b) {
					if b[i] == '*' && b[i+1] == '/' {
						i += 2
						found = true
						break
					}
					i++
				}
				if found {
					continue
				}
			}
		}
		break
	}

	remaining := b[i:]
	if len(remaining) < 12 {
		return b
	}

	// Check for "use strict" or 'use strict'
	if (remaining[0] == '"' || remaining[0] == '\'') &&
		remaining[1] == 'u' && remaining[2] == 's' && remaining[3] == 'e' && remaining[4] == ' ' &&
		remaining[5] == 's' && remaining[6] == 't' && remaining[7] == 'r' && remaining[8] == 'i' &&
		remaining[9] == 'c' && remaining[10] == 't' &&
		remaining[11] == remaining[0] {

		res := remaining[12:]
		if len(res) > 0 && res[0] == ';' {
			res = res[1:]
		}
		return res
	}

	return b
}
