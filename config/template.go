package config

import "strings"

// Render performs placeholder substitution for {{key}} style templates.
func Render(tmpl string, data map[string]string) string {
	out := tmpl
	for key, value := range data {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return out
}
