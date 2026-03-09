package utility

import "strings"

func ExtractEcosystem(purl string) string {
	if !strings.HasPrefix(purl, "pkg:") {
		return ""
	}
	without := strings.TrimPrefix(purl, "pkg:")
	parts := strings.SplitN(without, "/", 2)
	if len(parts) < 1 {
		return ""
	}
	return parts[0]
}
