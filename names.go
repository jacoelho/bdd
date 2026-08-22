package bdd

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tags))
	for _, tag := range tags {
		parts = append(parts, "@"+tag)
	}
	return strings.Join(parts, "_")
}

func appendTags(existing []string, tags ...string) []string {
	out := append([]string(nil), existing...)
	for _, tag := range tags {
		seen := false
		for _, current := range out {
			if current == tag {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, tag)
		}
	}
	sort.Strings(out)
	return out
}

func normalizeTag(tag string) (string, error) {
	if tag == "" {
		return "", fmt.Errorf("tag is empty")
	}
	if strings.HasPrefix(tag, "@") {
		return "", fmt.Errorf("tag %q must omit the leading @", tag)
	}

	runes := []rune(tag)
	var out strings.Builder
	segmentStart := true
	for _, r := range runes {
		switch {
		case r == '-':
			if segmentStart {
				return "", fmt.Errorf("tag %q contains an empty segment", tag)
			}
			out.WriteRune(r)
			segmentStart = true
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			out.WriteRune(unicode.ToLower(r))
			segmentStart = false
		case unicode.IsMark(r) && !segmentStart:
			out.WriteRune(r)
		default:
			return "", fmt.Errorf("tag %q contains invalid rune %q", tag, r)
		}
	}
	if segmentStart {
		return "", fmt.Errorf("tag %q contains an empty segment", tag)
	}
	return out.String(), nil
}

func slugName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name is blank")
	}

	var out strings.Builder
	inToken := false
	separator := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			if separator && out.Len() > 0 {
				out.WriteByte('-')
			}
			out.WriteRune(unicode.ToLower(r))
			inToken = true
			separator = false
		case unicode.IsMark(r) && inToken:
			out.WriteRune(r)
		case r == ' ' || unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) || unicode.IsControl(r):
			if out.Len() > 0 {
				separator = true
			}
			inToken = false
		default:
			if out.Len() > 0 {
				separator = true
			}
			inToken = false
		}
	}

	result := strings.Trim(out.String(), "-")
	if result == "" {
		return "", fmt.Errorf("name %q contains no letters or numbers", name)
	}
	return result, nil
}
