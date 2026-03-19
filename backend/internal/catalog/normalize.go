package catalog

import (
	"regexp"
	"strings"
)

var nonDigitsRegex = regexp.MustCompile(`\D+`)
var multiSpaceRegex = regexp.MustCompile(`\s+`)

func NormalizeGTIN(v string) string {
	v = strings.TrimSpace(strings.ToUpper(v))

	if v == "" {
		return ""
	}

	if v == "SEM GTIN" || v == "SEMGTIN" || v == "NO GTIN" {
		return ""
	}

	v = nonDigitsRegex.ReplaceAllString(v, "")

	if v == "" {
		return ""
	}

	// evita gtins obviamente ruins
	if allSameDigit(v) {
		return ""
	}

	return v
}

func NormalizeDescription(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	if v == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		".", " ",
		",", " ",
		";", " ",
		":", " ",
		"/", " ",
		"\\", " ",
		"-", " ",
		"_", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
		"\"", " ",
		"'", " ",
	)

	v = replacer.Replace(v)
	v = multiSpaceRegex.ReplaceAllString(v, " ")
	return strings.TrimSpace(v)
}

func allSameDigit(v string) bool {
	if len(v) == 0 {
		return false
	}

	first := v[0]
	for i := 1; i < len(v); i++ {
		if v[i] != first {
			return false
		}
	}
	return true
}
