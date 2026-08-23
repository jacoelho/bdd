package bdd

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
)

type crumb struct {
	kind string
	slug string
	raw  string
}

func (c crumb) name() string {
	return c.kind + "-" + c.slug
}

type featurePlan[S any] struct {
	setup  func(testing.TB) S
	leaves []*leaf[S]
	seen   map[string]string
	names  map[string]string
	rules  map[string]string
}

type leaf[S any] struct {
	name       string
	key        string
	breadcrumb string
	diagnostic string
	meta       meta
	before     []Step[S]
	background []step[S]
	build      func(*ScenarioSpec[S])
	state      *featurePlan[S]
	source     string
}

// Suite is a typed BDD declaration scope. Its fields are intentionally
// private; use Feature, Rule, Scenario, Outline, hooks, and
// Background to declare behavior.
type Suite[S any] struct {
	feature       *featurePlan[S]
	owner         *testing.T
	tags          []string
	breadcrumbs   []crumb
	background    []step[S]
	before        []Step[S]
	hasBackground bool
	children      bool
	sealed        bool
}

var featureClaims = struct {
	sync.Mutex
	owners map[*testing.T]string
}{owners: make(map[*testing.T]string)}

// Feature creates one top-level BDD grouping subtest. Each selected active
// Scenario or outline example is registered as one direct child and receives
// exactly one fresh state from setup. Setup runs after a declared Parallel
// resumes and before hooks and steps; it receives that leaf's testing.TB.
// Setup is not called for filtered or disposed leaves and may return a zero
// value, including a nil pointer. Its Cleanup, Error, Fatal, Skip, and panic
// behavior follows the standard testing package. Setup must be nonnil. A nil
// Feature builder is legal only when a disposition option makes the Feature
// non-executable.
func Feature[S any](t *testing.T, name string, setup func(testing.TB) S, build func(*Suite[S]), opts ...Option) {
	where := declarationSource()
	if t == nil {
		panic("bdd: Feature requires a nonnil *testing.T")
	}
	t.Helper()
	if strings.Contains(t.Name(), "/") {
		t.Fatalf("bdd: %s: Feature must be called from a direct top-level test", where)
	}
	if setup == nil {
		t.Fatalf("bdd: %s: Feature %q has a nil setup callback", where, name)
	}
	featureSlug, err := slugName(name)
	if err != nil {
		t.Fatalf("bdd: %s: invalid Feature name %q: %v", where, name, err)
	}
	m, err := (meta{}).with(opts...)
	if err != nil {
		t.Fatalf("bdd: %s: invalid Feature options: %v", where, err)
	}
	if build == nil && !m.hasDisposition() {
		t.Fatalf("bdd: %s: Feature %q has a nil builder", where, name)
	}
	if err := claimFeature(t, where); err != nil {
		t.Fatalf("bdd: %s: %v", where, err)
	}

	t.Run("feature-"+featureSlug, func(featureT *testing.T) {
		if m.hasDisposition() {
			terminalName := "feature-" + featureSlug
			if tags := formatTags(sortedTags(m.tags)); tags != "" {
				terminalName = tags + "__" + terminalName
			}
			featureT.Run(terminalName, func(terminal *testing.T) {
				terminal.Skip(featureDisposition(name, m))
			})
			return
		}

		plan := &featurePlan[S]{
			setup: setup,
			seen:  make(map[string]string),
			names: make(map[string]string),
			rules: make(map[string]string),
		}
		suite := &Suite[S]{
			feature: plan,
			owner:   featureT,
			tags:    slices.Clone(m.tags),
		}
		func() {
			defer suite.seal()
			build(suite)
		}()
		if featureT.Failed() || featureT.Skipped() {
			return
		}
		if len(plan.leaves) == 0 {
			featureT.Fatalf("bdd: %s: Feature %q must contain at least one Scenario or Outline example", where, name)
		}
		for _, leaf := range plan.leaves {
			featureT.Run(leaf.name, leaf.run)
		}
	})
}

func claimFeature(t *testing.T, where string) error {
	featureClaims.Lock()
	defer featureClaims.Unlock()
	if previous, ok := featureClaims.owners[t]; ok {
		return fmt.Errorf("a Feature is already registered for %s (previous declaration at %s)", t.Name(), previous)
	}
	featureClaims.owners[t] = where
	t.Cleanup(func() {
		featureClaims.Lock()
		delete(featureClaims.owners, t)
		featureClaims.Unlock()
	})
	return nil
}

// Rule creates a logical rule breadcrumb. It does not create a testing
// subtest, so all descendant scenarios remain direct Feature children. A nil
// builder is legal only when a disposition option makes the Rule non-executable.
func (s *Suite[S]) Rule(name string, build func(*Suite[S]), opts ...Option) {
	s.rule(name, build, declarationSource(), opts...)
}

// Scenario declares one atomic scenario leaf. Its builder runs lazily inside
// the selected leaf, after Go's -run filtering has selected it. A nil builder
// is legal only for a disposed Scenario.
func (s *Suite[S]) Scenario(name string, build func(*ScenarioSpec[S]), opts ...Option) {
	where := declarationSource()
	s.declareChild()
	slug, err := slugName(name)
	if err != nil {
		s.failfAt(where, "invalid Scenario name %q: %v", name, err)
	}
	m, err := s.nodeMeta(opts...)
	if err != nil {
		s.failfAt(where, "invalid Scenario %q options: %v", name, err)
	}
	if build == nil && !m.hasDisposition() {
		s.failfAt(where, "Scenario %q has a nil builder", name)
	}
	crumbs := append(slices.Clone(s.breadcrumbs), crumb{kind: "scenario", slug: slug, raw: name})
	s.addLeaf(&leaf[S]{
		name:       s.leafName(crumbs, m.tags),
		key:        breadcrumbKey(crumbs),
		breadcrumb: rawBreadcrumb(crumbs),
		meta:       m,
		before:     slices.Clone(s.before),
		background: slices.Clone(s.background),
		build:      build,
		state:      s.feature,
		diagnostic: dispositionDiagnostic(rawBreadcrumb(crumbs), m),
		source:     where,
	})
}

// Background adds a visible setup operation and returns a private capability
// exposing only And and But. Reassign every returned value when storing the
// capability. Copying a value is harmless; a stale value used while the Suite
// builder is active fails through the owning declaration, while any value used
// after the builder returns panics with the owner-sealed diagnostic.
func (s *Suite[S]) Background(text string, fn Step[S]) backgroundChain[S] {
	where := declarationSource()
	s.ensureOpen()
	s.appendBackground(where, kindBackground, text, fn)
	s.hasBackground = true
	return s.newBackgroundChain(where)
}

// Before adds a hidden setup hook inherited by descendant scenarios.
func (s *Suite[S]) Before(fn Step[S]) {
	where := declarationSource()
	s.ensureOpen()
	if s.children {
		s.failfAt(where, "Before must be declared before child scenarios")
	}
	if s.hasBackground {
		s.failfAt(where, "Before must be declared before Background")
	}
	if fn == nil {
		s.failfAt(where, "Before has a nil step")
	}
	s.before = append(s.before, fn)
}

func (s *Suite[S]) newBackgroundChain(issued string) backgroundChain[S] {
	return backgroundChain[S]{core: chainCore[S]{
		expected: len(s.background),
		issued:   issued,
		advance:  s.appendBackgroundFromChain,
	}}
}

func (s *Suite[S]) appendBackgroundFromChain(expected int, issued, where string, kind stepKind, text string, fn Step[S]) int {
	// Check ownership before generation so a token used after this Suite's
	// builder returns reports the stable sealed-owner diagnostic.
	s.ensureOpen()
	if len(s.background) != expected {
		s.failfAt(where, "stale Background chain token issued at %s; expected operation generation %d, found %d; assign every returned chain value", issued, expected, len(s.background))
	}
	s.appendBackground(where, kind, text, fn)
	return expected + 1
}

func (s *Suite[S]) appendBackground(where string, kind stepKind, text string, fn Step[S]) {
	s.ensureOpen()
	if s.children {
		s.failfAt(where, "Background must be declared before child scenarios")
	}
	if kind != kindBackground && !s.hasBackground {
		s.failfAt(where, "%s requires a local Background", kind)
	}
	if kind == kindBackground && s.hasBackground {
		s.failfAt(where, "Suite may declare only one local Background")
	}
	if fn == nil {
		s.failfAt(where, "%s %q has a nil step", kind, text)
	}
	if strings.TrimSpace(text) == "" {
		s.failfAt(where, "%s text is blank", kind)
	}
	s.background = append(s.background, newStep(kind, text, fn))
}

func (s *Suite[S]) rule(name string, build func(*Suite[S]), where string, opts ...Option) {
	s.declareChild()
	slug, err := slugName(name)
	if err != nil {
		s.failfAt(where, "invalid Rule name %q: %v", name, err)
	}
	m, err := s.nodeMeta(opts...)
	if err != nil {
		s.failfAt(where, "invalid Rule %q options: %v", name, err)
	}
	crumbs := append(slices.Clone(s.breadcrumbs), crumb{kind: "rule", slug: slug, raw: name})
	ruleKey := breadcrumbKey(crumbs)
	if previous, ok := s.feature.rules[ruleKey]; ok {
		s.failfAt(where, "duplicate Rule identity %q; previous declaration at %s, current declaration at %s", ruleKey, previous, where)
	}
	s.feature.rules[ruleKey] = where
	if m.hasDisposition() {
		s.addLeaf(&leaf[S]{
			name:       s.leafName(crumbs, m.tags),
			key:        breadcrumbKey(crumbs),
			breadcrumb: rawBreadcrumb(crumbs),
			meta:       m,
			before:     slices.Clone(s.before),
			background: slices.Clone(s.background),
			state:      s.feature,
			diagnostic: dispositionDiagnostic(rawBreadcrumb(crumbs), m),
			source:     where,
		})
		return
	}
	if build == nil {
		s.failfAt(where, "Rule %q has a nil builder", name)
	}

	child := &Suite[S]{
		feature:       s.feature,
		owner:         s.owner,
		tags:          slices.Clone(m.tags),
		breadcrumbs:   crumbs,
		background:    slices.Clone(s.background),
		before:        slices.Clone(s.before),
		hasBackground: false,
	}
	start := len(s.feature.leaves)
	func() {
		defer child.seal()
		build(child)
	}()
	if s.owner.Failed() || s.owner.Skipped() {
		return
	}
	if len(s.feature.leaves) == start {
		s.failfAt(where, "Rule %q must contain at least one Scenario or Outline example", name)
	}
}

func (s *Suite[S]) declareChild() {
	s.ensureOpen()
	s.children = true
}

func (s *Suite[S]) nodeMeta(opts ...Option) (meta, error) {
	m := meta{tags: slices.Clone(s.tags)}
	return m.with(opts...)
}

func (s *Suite[S]) addLeaf(leaf *leaf[S]) {
	where := leaf.source
	if previous, ok := s.feature.seen[leaf.key]; ok {
		s.failfAt(where, "duplicate semantic scenario identity %q; previous declaration at %s, current declaration at %s", leaf.key, previous, where)
	}
	if previous, ok := s.feature.names[leaf.name]; ok {
		s.failfAt(where, "duplicate generated test name %q; previous declaration at %s, current declaration at %s", leaf.name, previous, where)
	}
	s.feature.seen[leaf.key] = where
	s.feature.names[leaf.name] = where
	s.feature.leaves = append(s.feature.leaves, leaf)
}

func (s *Suite[S]) leafName(crumbs []crumb, tags []string) string {
	parts := make([]string, 0, len(crumbs))
	for _, crumb := range crumbs {
		parts = append(parts, crumb.name())
	}
	base := strings.Join(parts, "__")
	if tagPrefix := formatTags(sortedTags(tags)); tagPrefix != "" {
		return tagPrefix + "__" + base
	}
	return base
}

func breadcrumbKey(crumbs []crumb) string {
	parts := make([]string, 0, len(crumbs))
	for _, crumb := range crumbs {
		parts = append(parts, crumb.name())
	}
	return strings.Join(parts, "__")
}

func rawBreadcrumb(crumbs []crumb) string {
	parts := make([]string, 0, len(crumbs))
	for _, crumb := range crumbs {
		parts = append(parts, crumb.kind+": "+crumb.raw)
	}
	return strings.Join(parts, " > ")
}

func (s *Suite[S]) ensureOpen() {
	if s == nil {
		panic("bdd: Suite receiver is nil")
	}
	if s.feature == nil || s.owner == nil {
		panic("bdd: Suite receiver is zero")
	}
	if s.sealed {
		panic("bdd: Suite builder has returned")
	}
}

func (s *Suite[S]) seal() {
	s.sealed = true
}

func (s *Suite[S]) failfAt(where, format string, args ...any) {
	s.ensureOpen()
	if where != "" {
		format = where + ": " + format
	}
	s.owner.Fatalf("bdd: "+format, args...)
}

func (p *featurePlan[S]) runLeaf(t *testing.T, leaf *leaf[S]) {
	if leaf.meta.hasDisposition() {
		t.Skip(leaf.diagnostic)
	}
	t.Helper()
	t.Log(leaf.breadcrumb)
	spec := &ScenarioSpec[S]{
		owner:      t,
		before:     slices.Clone(leaf.before),
		background: slices.Clone(leaf.background),
	}
	func() {
		defer spec.seal()
		leaf.build(spec)
	}()
	if t.Failed() || t.Skipped() {
		return
	}
	if len(spec.steps) == 0 {
		t.Fatalf("bdd: %s: scenario %q must contain at least one visible operation", leaf.source, leaf.name)
	}
	if spec.parallel {
		t.Parallel()
	}
	t.Logf("Setup: %s", leaf.breadcrumb)
	state := p.setup(t)
	if t.Failed() || t.Skipped() {
		return
	}
	for _, hook := range spec.before {
		t.Logf("Before: %s", leaf.breadcrumb)
		hook(t, state)
		if t.Failed() || t.Skipped() {
			return
		}
	}
	for _, st := range spec.background {
		runStep(t, st, state)
		if t.Failed() || t.Skipped() {
			return
		}
	}
	for _, st := range spec.steps {
		runStep(t, st, state)
		if t.Failed() || t.Skipped() {
			return
		}
	}
}

func featureDisposition(name string, m meta) string {
	verb := "skipped"
	if m.disposition == dispositionPending {
		verb = "pending"
	}
	return fmt.Sprintf("bdd: Feature %q %s: %s", name, verb, m.reason)
}

func dispositionDiagnostic(breadcrumb string, m meta) string {
	if !m.hasDisposition() {
		return ""
	}
	verb := "skipped"
	if m.disposition == dispositionPending {
		verb = "pending"
	}
	return fmt.Sprintf("bdd: %s %s: %s", breadcrumb, verb, m.reason)
}

func (l *leaf[S]) run(t *testing.T) {
	l.state.runLeaf(t, l)
}
