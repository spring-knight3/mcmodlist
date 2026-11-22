package jar

import "strings"

func normalizeAuthors(value any) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	var walk func(any)
	walk = func(v any) {
		switch t := v.(type) {
		case nil:
		case string:
			for _, part := range strings.FieldsFunc(t, func(r rune) bool { return r == ',' || r == ';' }) {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				if _, ok := seen[part]; !ok {
					seen[part] = struct{}{}
					out = append(out, part)
				}
			}
		case []any:
			for _, item := range t {
				walk(item)
			}
		case map[string]any:
			for _, key := range []string{"name", "author", "authors", "authorList"} {
				if child, ok := t[key]; ok {
					walk(child)
					return
				}
			}
		}
	}
	walk(value)
	return out
}
