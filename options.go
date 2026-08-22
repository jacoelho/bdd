package bdd

import (
	"fmt"
	"runtime"
	"slices"
	"sort"
	"strings"
)

// Option customizes a Feature, Rule, Context, Scenario, Outline, or Example.
//
// Option's fields are intentionally private. Construct options with Tags,
// Skip, or Pending; the zero value is invalid. Case constructs an Example,
// not an Option.
type Option struct {
	kind   optionKind
	tags   []string
	reason string
	where  string
	err    string
}

type optionKind uint8

const (
	optionInvalid optionKind = iota
	optionTags
	optionSkip
	optionPending
)

type meta struct {
	tags        []string
	disposition disposition
	reason      string
}

type disposition uint8

const (
	dispositionNone disposition = iota
	dispositionSkip
	dispositionPending
)

func declarationSource() string {
	// The helper is called directly by each exported declaration boundary;
	// capture the user's call site once and carry it through compilation.
	if _, file, line, ok := runtime.Caller(2); ok {
		return fmt.Sprintf("%s:%d", file, line)
	}
	return "unknown location"
}

// Tags annotates a Feature, Rule, Context, Scenario, Outline, or Example.
// Tags are inherited by children and added to generated test names as @tag, so
// ordinary go test -run filtering can select them. Tags are Unicode strings
// made of one or more hyphen-separated letter/number segments; callers omit
// the leading @.
func Tags(tags ...string) Option {
	o := Option{kind: optionTags, where: declarationSource(), tags: slices.Clone(tags)}
	if len(tags) == 0 {
		o.err = "tags requires at least one tag"
		return o
	}
	for i, tag := range tags {
		normalized, err := normalizeTag(tag)
		if err != nil {
			o.err = err.Error()
			return o
		}
		o.tags[i] = normalized
	}
	return o
}

// Skip marks a Feature, Rule, Context, Scenario, Outline, or Example as
// skipped. Pass it to Case for example-specific disposition.
func Skip(reason string) Option {
	o := Option{kind: optionSkip, reason: strings.TrimSpace(reason), where: declarationSource()}
	if o.reason == "" {
		o.err = "skip requires a nonblank reason"
	}
	return o
}

// Pending marks a Feature, Rule, Context, Scenario, Outline, or Example as
// pending. Pass it to Case for example-specific disposition. Pending nodes are
// represented as skipped tests by Go's testing package.
func Pending(reason string) Option {
	o := Option{kind: optionPending, reason: strings.TrimSpace(reason), where: declarationSource()}
	if o.reason == "" {
		o.err = "pending requires a nonblank reason"
	}
	return o
}

func (m meta) with(opts ...Option) (meta, error) {
	next := meta{
		tags:        slices.Clone(m.tags),
		disposition: m.disposition,
		reason:      m.reason,
	}
	for _, opt := range opts {
		if opt.kind == optionInvalid || opt.err != "" {
			return meta{}, fmt.Errorf("invalid option at %s: %s", opt.where, optionError(opt))
		}
		switch opt.kind {
		case optionTags:
			next.tags = appendTags(next.tags, opt.tags...)
		case optionSkip, optionPending:
			if next.disposition != dispositionNone {
				return meta{}, fmt.Errorf("multiple disposition options")
			}
			next.reason = opt.reason
			if opt.kind == optionSkip {
				next.disposition = dispositionSkip
			} else {
				next.disposition = dispositionPending
			}
		default:
			return meta{}, fmt.Errorf("invalid option at %s: unknown option kind", opt.where)
		}
	}
	return next, nil
}

func optionError(opt Option) string {
	if opt.err != "" {
		return opt.err
	}
	return "zero option"
}

func (m meta) hasDisposition() bool {
	return m.disposition != dispositionNone
}

func sortedTags(tags []string) []string {
	out := slices.Clone(tags)
	sort.Strings(out)
	return out
}
