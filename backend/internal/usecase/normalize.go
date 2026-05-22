package usecase

import (
	"regexp"
	"strings"
	"unicode"

	packageurl "github.com/package-url/packageurl-go"
)

type NormalizedIdentifier struct {
	RawInput    string
	Ecosystem   string
	Namespace   string
	PackageName string
	Version     string
	Words       []string
}

var (
	pseudoVersionRe = regexp.MustCompile(`^(v\d+\.\d+\.\d+)-\d+\.\d{14}-[0-9a-fA-F]{12}$`)

	genericTokens = map[string]bool{
		"library": true, "package": true, "module": true, "framework": true,
		"plugin": true, "packages": true, "multiple": true, "versions": true,
		"unknown": true, "pending": true, "tools": true, "utils": true,
		"core": true, "base": true, "common": true, "shared": true,
		"javascript": true,
	}
)

func Normalize(purl, name string) NormalizedIdentifier {
	if purl == "" && name == "" {
		return NormalizedIdentifier{}
	}

	rawInput := purl
	if rawInput == "" {
		rawInput = name
	}

	decoded := percentDecodePURL(purl)

	if shouldRejectInput(decoded, name) {
		return NormalizedIdentifier{RawInput: rawInput}
	}

	ecosystem, namespace, packageName, version, purlParsed := parsePURL(decoded)

	if !purlParsed {
		ecosystem = ecosystemFromString(name)
	}

	if packageName == "" {
		src := name
		if src == "" {
			src = decoded
		}
		packageName = nameFromRawString(src)
	}

	return NormalizedIdentifier{
		RawInput:    rawInput,
		Ecosystem:   strings.ToLower(ecosystem),
		Namespace:   namespace,
		PackageName: strings.ToLower(packageName),
		Version:     strings.ToLower(cleanVersion(version)),
		Words:       searchTerms(packageName, namespace, name),
	}
}

func shouldRejectInput(decoded, name string) bool {
	if isPlaceholder(name) {
		return true
	}
	if decoded == "" {
		return false
	}
	p, err := packageurl.FromString(decoded)
	if err != nil {
		return false
	}
	if isPlaceholder(p.Name) {
		return true
	}
	return p.Name == "*" && p.Namespace == ""
}

func isPlaceholder(s string) bool {
	return len(s) > 1 && s[0] == '<' && s[len(s)-1] == '>'
}

// net/url is intentionally avoided — it restructures the opaque pkg: scheme.
func percentDecodePURL(purl string) string {
	purl = strings.ReplaceAll(purl, "%40", "@")
	purl = strings.ReplaceAll(purl, "%2B", "+")
	purl = strings.ReplaceAll(purl, "%2b", "+")
	purl = strings.ReplaceAll(purl, "%2F", "/")
	purl = strings.ReplaceAll(purl, "%2f", "/")
	return purl
}

func parsePURL(decoded string) (ecosystem, namespace, packageName, version string, ok bool) {
	if decoded == "" {
		return
	}
	p, err := packageurl.FromString(decoded)
	if err != nil {
		return
	}
	return p.Type, extractNamespace(p.Namespace), packageNameFromPURL(p), p.Version, true
}

func extractNamespace(raw string) string {
	if raw == "" {
		return ""
	}
	raw = strings.TrimPrefix(raw, "@")
	parts := strings.Split(raw, "/")
	last := strings.TrimPrefix(parts[len(parts)-1], "@")
	return strings.ToLower(last)
}

func packageNameFromPURL(p packageurl.PackageURL) string {
	if isMajorVersionSuffix(p.Name) || p.Name == "*" {
		parts := strings.Split(p.Namespace, "/")
		if len(parts) > 0 {
			return strings.TrimPrefix(parts[len(parts)-1], "@")
		}
	}
	return p.Name
}

func isMajorVersionSuffix(s string) bool {
	return len(s) > 1 && s[0] == 'v' && unicode.IsDigit(rune(s[1]))
}

func ecosystemFromString(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lower, "pkg:npm") || strings.Contains(lower, "npmjs"):
		return "npm"
	case strings.HasPrefix(lower, "pkg:pypi"):
		return "pypi"
	case strings.HasPrefix(lower, "pkg:golang") || strings.Contains(lower, "golang.org"):
		return "golang"
	default:
		return ""
	}
}

func nameFromRawString(s string) string {
	if s == "" {
		return ""
	}
	if idx := strings.LastIndex(s, "@"); idx > 0 {
		s = s[:idx]
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '/' || r == ':' || r == '@'
	})
	if len(parts) == 0 {
		return s
	}
	last := parts[len(parts)-1]
	if isMajorVersionSuffix(last) && len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return last
}

func cleanVersion(version string) string {
	if idx := strings.Index(version, "+"); idx > 0 {
		version = version[:idx]
	}
	if m := pseudoVersionRe.FindStringSubmatch(version); m != nil {
		return m[1]
	}
	return version
}

func splitIntoTokens(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return unicode.IsSpace(r) || r == '/' || r == '-' || r == '_' || r == '.' || r == '@'
	})
}

func searchTerms(packageName, namespace, name string) []string {
	seen := make(map[string]bool)
	var terms []string
	for _, src := range []string{packageName, namespace, name} {
		for _, tok := range splitIntoTokens(src) {
			if len(tok) < 3 || genericTokens[tok] || seen[tok] {
				continue
			}
			seen[tok] = true
			terms = append(terms, tok)
		}
	}
	return terms
}
