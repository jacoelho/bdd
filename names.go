package bdd

import (
	"fmt"
	"slices"
	"strings"
)

func groupName(kind, name string, m meta) string {
	name = cleanName(name)
	tags := formatTags(m.tags)
	if tags == "" {
		return fmt.Sprintf("%s: %s", kind, name)
	}
	return fmt.Sprintf("%s: %s %s", kind, tags, name)
}

func scenarioName(name string, m meta) string {
	return groupName("Scenario", name, m)
}

func outlineName(name string, m meta) string {
	return groupName("Scenario Outline", name, m)
}

func stepName(kind stepKind, text string) string {
	return fmt.Sprintf("%s: %s", kind, cleanName(text))
}

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = normalizeTag(tag)
		if tag != "" {
			parts = append(parts, "@"+tag)
		}
	}
	return strings.Join(parts, " ")
}

func appendTags(existing []string, tags ...string) []string {
	out := slices.Clone(existing)
	for _, tag := range tags {
		tag = normalizeTag(tag)
		if tag == "" {
			continue
		}
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
	return out
}

func normalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	tag = strings.TrimPrefix(tag, "@")
	tag = strings.Join(strings.Fields(tag), "-")
	tag = strings.ReplaceAll(tag, "/", "-")
	return tag
}

func cleanName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "unnamed"
	}
	return s
}
