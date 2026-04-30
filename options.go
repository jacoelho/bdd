package bdd

import (
	"slices"
	"strings"
)

// Option customizes a feature, rule, scenario, or outline.
type Option interface {
	apply(*meta)
}

type optionFunc func(*meta)

func (f optionFunc) apply(m *meta) {
	f(m)
}

type meta struct {
	tags          []string
	skipSet       bool
	skipReason    string
	pendingSet    bool
	pendingReason string
}

// Tags annotates a feature, rule, scenario, or outline. Tags are inherited by
// children and added to generated test names as @tag, so ordinary go test -run
// filtering can select them.
func Tags(tags ...string) Option {
	return optionFunc(func(m *meta) {
		m.tags = appendTags(m.tags, tags...)
	})
}

// Skip marks a feature, rule, scenario, or outline as skipped.
func Skip(reason string) Option {
	return optionFunc(func(m *meta) {
		m.skipSet = true
		m.skipReason = strings.TrimSpace(reason)
	})
}

// Pending marks a feature, rule, scenario, or outline as pending. Pending nodes
// are represented as skipped tests by Go's testing package.
func Pending(reason string) Option {
	return optionFunc(func(m *meta) {
		m.pendingSet = true
		m.pendingReason = strings.TrimSpace(reason)
	})
}

func (m meta) with(opts ...Option) meta {
	next := meta{
		tags:          slices.Clone(m.tags),
		skipSet:       m.skipSet,
		skipReason:    m.skipReason,
		pendingSet:    m.pendingSet,
		pendingReason: m.pendingReason,
	}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&next)
		}
	}
	return next
}

func (m meta) withTags(tags ...string) meta {
	next := m.with()
	next.tags = appendTags(next.tags, tags...)
	return next
}

func (m meta) skipMessage() string {
	if m.skipSet {
		if m.skipReason == "" {
			return "skipped"
		}
		return m.skipReason
	}
	if m.pendingSet {
		if m.pendingReason == "" {
			return "pending"
		}
		return "pending: " + m.pendingReason
	}
	return ""
}
