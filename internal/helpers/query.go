package helpers

import "strings"

func NormalizeQuery(query string) string {
	lines := strings.Split(query, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, " ")
}
