package parsefiles

import (
	"bufio"
	"os"
	"strings"
)

// EnvValue returns VALUE for KEY= in a .env style file (first match).
func EnvValue(path, key string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	prefix := key + "="
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

// TomlField returns the scalar value after `key = ` on a line (handles inline # comments and "quoted" strings).
func TomlField(path, key string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	prefix := key + " = "
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			rest := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			rest = stripTomlLineComment(rest)
			return parseTomlScalar(rest)
		}
	}
	return ""
}

// stripTomlLineComment removes a trailing # comment when it is not inside double quotes.
func stripTomlLineComment(s string) string {
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && inQuote && i+1 < len(s) {
			i++
			continue
		}
		if c == '"' {
			inQuote = !inQuote
			continue
		}
		if c == '#' && !inQuote {
			return strings.TrimSpace(s[:i])
		}
	}
	return strings.TrimSpace(s)
}

// parseTomlScalar parses a single TOML string or bare token (first line segment).
func parseTomlScalar(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if raw[0] != '"' {
		fields := strings.Fields(raw)
		if len(fields) > 0 {
			return fields[0]
		}
		return raw
	}
	var b strings.Builder
	for i := 1; i < len(raw); i++ {
		c := raw[i]
		if c == '\\' && i+1 < len(raw) {
			i++
			b.WriteByte(raw[i])
			continue
		}
		if c == '"' {
			return b.String()
		}
		b.WriteByte(c)
	}
	return b.String()
}
