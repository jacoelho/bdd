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
	// seen, names, and rules map an identity to its declaration source.
	seen  map[string]string
	names map[string]string
	rules map[string]string
}

func newFeaturePlan[S any](setup func(testing.TB) S) *featurePlan[S] {
	return &featurePlan[S]{
		setup: setup,
		seen:  make(map[string]string),
		names: make(map[string]string),
		rules: make(map[string]string),
	}
}

// leaf is one direct Feature child: an active scenario, or a placeholder for
// a disposed Rule, Scenario, or Outline example. build is nil only when meta
// carries a disposition; before and background are the inherited operations
// of an active leaf.
type leaf[S any] struct {
	name       string
	breadcrumb string
	source     string
	meta       meta
	before     []step[S]
	background []step[S]
	build      func(*ScenarioSpec[S])
}

// suitePhase is the declaration state of a Suite. suitePhase.next owns which
// declarations each phase accepts (suiteOrder); Rule suspends and resumes the
// Suite around its nested builder, and seal ends it.
type suitePhase uint8

const (
	suiteUnbuilt    suitePhase = iota // zero value: not created by the framework
	suiteHooks                        // accepts Before, Background, and children
	suiteBackground                   // Background declared: accepts its And/But chain and children
	suiteChildren                     // a child was declared: accepts children only
	suiteSuspended                    // a nested Rule builder is running: accepts nothing
	suiteSealed                       // the builder returned
)

// suiteDeclaration classifies the declarations a Suite accepts.
type suiteDeclaration uint8

const (
	declareHook         suiteDeclaration = iota // Before
	declareBackground                           // Background
	declareContinuation                         // And or But on the Background chain
	declareChild                                // Scenario, Outline, or Rule
)

const suiteOrder = "a Suite declares Before hooks, then one Background chain, then Scenario, Outline, and Rule children"

// next returns the phase that follows declaration d, or why d is illegal in
// phase p. Unbuilt and sealed Suites never reach it: ensureOpen panics first.
func (p suitePhase) next(d suiteDeclaration) (suitePhase, string) {
	switch p {
	case suiteHooks:
		switch d {
		case declareHook:
			return suiteHooks, ""
		case declareBackground:
			return suiteBackground, ""
		case declareChild:
			return suiteChildren, ""
		}
	case suiteBackground:
		switch d {
		case declareHook:
			return p, "Before must be declared before Background; " + suiteOrder
		case declareBackground:
			return p, "Suite may declare only one local Background; continue the first with And or But on its returned chain"
		case declareContinuation:
			return suiteBackground, ""
		case declareChild:
			return suiteChildren, ""
		}
	case suiteChildren:
		switch d {
		case declareHook:
			return p, "Before must be declared before child scenarios; " + suiteOrder
		case declareBackground, declareContinuation:
			return p, "Background must be declared before child scenarios; " + suiteOrder
		case declareChild:
			return suiteChildren, ""
		}
	case suiteSuspended:
		return p, "this Suite is suspended while its nested Rule builder runs; declare on the *Suite passed to that builder, not on an enclosing Suite"
	}
	// A continuation needs a chain, and only Background issues one.
	panic(fmt.Sprintf("bdd: unreachable Suite transition: phase %d, declaration %d", p, d))
}

// Suite is a typed BDD declaration scope. Its fields are intentionally
// private; use Feature, Rule, Scenario, Outline, hooks, and
// Background to declare behavior.
//
// Declare in this order: Before hooks, then one Background chain, then
// Scenario, Outline, and Rule children. While a nested Rule builder runs, its
// enclosing Suites accept no declarations; a Suite is sealed when its builder
// returns.
type Suite[S any] struct {
	feature     *featurePlan[S]
	owner       *testing.T
	phase       suitePhase
	tags        []string
	breadcrumbs []crumb
	before      []step[S]
	background  []step[S]
}

func newSuite[S any](feature *featurePlan[S], owner *testing.T, tags []string) *Suite[S] {
	return &Suite[S]{feature: feature, owner: owner, phase: suiteHooks, tags: tags}
}

var featureClaims = struct {
	sync.Mutex
	owners map[*testing.T]string
}{owners: make(map[*testing.T]string)}

// Feature creates one top-level BDD grouping subtest. Each selected active
// Scenario or outline example is registered as one direct child and receives
// exactly one fresh state from setup. Setup runs after a declared Parallel
// resumes and before hooks and steps; it receives that child's testing.TB.
// Setup is not called for filtered or disposed children and may return a zero
// value, including a nil pointer. Its Cleanup, Error, Fatal, Skip, and panic
// behavior follows the standard testing package. Setup must be nonnil. A nil
// Feature builder is legal only when Skip or Pending disposes the Feature.
//
// Feature must be called at most once, directly from a top-level Test
// function.
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

	// Every framework frame between a log call and the user's test function
	// is marked as a helper, so framework output is attributed to the
	// Feature call site rather than to this package.
	t.Run("feature-"+featureSlug, func(featureT *testing.T) {
		featureT.Helper()
		if m.hasDisposition() {
			terminalName := "feature-" + featureSlug
			if tags := formatTags(sortedTags(m.tags)); tags != "" {
				terminalName = tags + "__" + terminalName
			}
			featureT.Run(terminalName, func(terminal *testing.T) {
				terminal.Helper()
				terminal.Skip(m.diagnostic(fmt.Sprintf("Feature %q", name)))
			})
			return
		}

		plan := newFeaturePlan(setup)
		suite := newSuite(plan, featureT, slices.Clone(m.tags))
		func() {
			defer suite.seal()
			build(suite)
		}()
		if halted(featureT) {
			return
		}
		if len(plan.leaves) == 0 {
			featureT.Fatalf("bdd: %s: Feature %q must contain at least one Scenario or Outline example", where, name)
		}
		for _, leaf := range plan.leaves {
			featureT.Run(leaf.name, func(leafT *testing.T) {
				leafT.Helper()
				plan.runLeaf(leafT, leaf)
			})
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

// Before adds a hidden setup hook inherited by descendant scenarios. It must
// be declared before this Suite's Background and children.
func (s *Suite[S]) Before(fn Step[S]) {
	where := declarationSource()
	s.declare(where, "Suite.Before", declareHook)
	if fn == nil {
		s.failfAt(where, "Before has a nil step")
	}
	s.before = append(s.before, step[S]{kind: kindBefore, source: where, fn: fn})
}

// Background adds a visible setup operation and returns a BackgroundChain
// exposing only And and But. A Suite declares at most one Background, after
// its Before hooks and before its children. Reassign every returned value
// when storing the chain. Copying a chain is harmless. While the Suite
// builder is active, using a chain that is not the latest fails the Suite's
// test; any chain used after the builder returns panics with the sealed-scope
// diagnostic.
func (s *Suite[S]) Background(text string, fn Step[S]) BackgroundChain[S] {
	where := declarationSource()
	s.declare(where, "Suite.Background", declareBackground)
	s.appendBackground(where, kindBackground, text, fn)
	return s.chain(where)
}

func (s *Suite[S]) continueBackground(token BackgroundChain[S], where string, kind stepKind, text string, fn Step[S]) BackgroundChain[S] {
	// Ownership is checked before the generation so a chain used after this
	// Suite's builder returned always reports the sealed-scope diagnostic.
	operation := "BackgroundChain." + string(kind)
	s.ensureOpen(where, operation)
	if token.generation != len(s.background) {
		s.failfAt(where, "stale Background chain token issued at %s; expected operation generation %d, found %d; assign every returned chain value", token.issued, token.generation, len(s.background))
	}
	s.declare(where, operation, declareContinuation)
	s.appendBackground(where, kind, text, fn)
	return s.chain(where)
}

func (s *Suite[S]) chain(issued string) BackgroundChain[S] {
	return BackgroundChain[S]{suite: s, generation: len(s.background), issued: issued}
}

func (s *Suite[S]) appendBackground(where string, kind stepKind, text string, fn Step[S]) {
	if fn == nil {
		s.failfAt(where, "%s %q has a nil step", kind, text)
	}
	if strings.TrimSpace(text) == "" {
		s.failfAt(where, "%s text is blank", kind)
	}
	s.background = append(s.background, step[S]{kind: kind, text: text, source: where, fn: fn})
}

// Scenario declares one atomic scenario, run as one direct Feature child. Its
// builder runs lazily inside that child, only after go test -run has selected
// it. A nil builder is legal only when Skip or Pending disposes the Scenario.
func (s *Suite[S]) Scenario(name string, build func(*ScenarioSpec[S]), opts ...Option) {
	where := declarationSource()
	s.declare(where, "Suite.Scenario", declareChild)
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
	s.addLeaf(where, s.crumbs("scenario", slug, name), m, build)
}

// Rule groups scenarios under a rule breadcrumb in the generated test name.
// It does not create a testing subtest, so all descendant scenarios remain
// direct Feature children. A nil builder is legal only when Skip or Pending
// disposes the Rule. While build runs, declare on its *Suite parameter: the
// enclosing Suites are suspended until it returns.
func (s *Suite[S]) Rule(name string, build func(*Suite[S]), opts ...Option) {
	where := declarationSource()
	s.declare(where, "Suite.Rule", declareChild)
	slug, err := slugName(name)
	if err != nil {
		s.failfAt(where, "invalid Rule name %q: %v", name, err)
	}
	m, err := s.nodeMeta(opts...)
	if err != nil {
		s.failfAt(where, "invalid Rule %q options: %v", name, err)
	}
	crumbs := s.crumbs("rule", slug, name)
	ruleKey := breadcrumbKey(crumbs)
	if previous, ok := s.feature.rules[ruleKey]; ok {
		s.failfAt(where, "duplicate Rule identity %q; previous declaration at %s, current declaration at %s", ruleKey, previous, where)
	}
	s.feature.rules[ruleKey] = where
	if m.hasDisposition() {
		s.addLeaf(where, crumbs, m, nil)
		return
	}
	if build == nil {
		s.failfAt(where, "Rule %q has a nil builder", name)
	}

	child := newSuite(s.feature, s.owner, slices.Clone(m.tags))
	child.breadcrumbs = crumbs
	child.before = slices.Clone(s.before)
	child.background = slices.Clone(s.background)
	start := len(s.feature.leaves)
	s.phase = suiteSuspended
	func() {
		defer func() {
			child.seal()
			s.phase = suiteChildren
		}()
		build(child)
	}()
	if halted(s.owner) {
		return
	}
	if len(s.feature.leaves) == start {
		s.failfAt(where, "Rule %q must contain at least one Scenario or Outline example", name)
	}
}

// declare applies the declaration grammar: it fails the owning test when d is
// illegal in the current phase and otherwise advances the phase.
func (s *Suite[S]) declare(where, operation string, d suiteDeclaration) {
	s.ensureOpen(where, operation)
	next, problem := s.phase.next(d)
	if problem != "" {
		s.failfAt(where, "%s", problem)
	}
	s.phase = next
}

func (s *Suite[S]) nodeMeta(opts ...Option) (meta, error) {
	m := meta{tags: slices.Clone(s.tags)}
	return m.with(opts...)
}

func (s *Suite[S]) crumbs(kind, slug, raw string) []crumb {
	return append(slices.Clone(s.breadcrumbs), crumb{kind: kind, slug: slug, raw: raw})
}

func (s *Suite[S]) addLeaf(where string, crumbs []crumb, m meta, build func(*ScenarioSpec[S])) {
	key := breadcrumbKey(crumbs)
	name := leafName(key, m.tags)
	if previous, ok := s.feature.seen[key]; ok {
		s.failfAt(where, "duplicate semantic scenario identity %q; previous declaration at %s, current declaration at %s", key, previous, where)
	}
	if previous, ok := s.feature.names[name]; ok {
		s.failfAt(where, "duplicate generated test name %q; previous declaration at %s, current declaration at %s", name, previous, where)
	}
	s.feature.seen[key] = where
	s.feature.names[name] = where
	leaf := &leaf[S]{
		name:       name,
		breadcrumb: rawBreadcrumb(crumbs),
		source:     where,
		meta:       m,
	}
	if !m.hasDisposition() {
		leaf.build = build
		leaf.before = slices.Clone(s.before)
		leaf.background = slices.Clone(s.background)
	}
	s.feature.leaves = append(s.feature.leaves, leaf)
}

func leafName(key string, tags []string) string {
	if tagPrefix := formatTags(sortedTags(tags)); tagPrefix != "" {
		return tagPrefix + "__" + key
	}
	return key
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

// ensureOpen panics unless s accepts declarations. A sealed scope panics
// rather than failing its owner because that test may already have finished.
func (s *Suite[S]) ensureOpen(where, operation string) {
	if s == nil {
		panic("bdd: Suite receiver is nil")
	}
	switch s.phase {
	case suiteUnbuilt:
		panic("bdd: Suite receiver is zero")
	case suiteSealed:
		panic(fmt.Sprintf("bdd: %s: %s called on a sealed Suite: its Feature or Rule builder has returned; declare inside that builder, and inside a Scenario builder declare on its *ScenarioSpec", where, operation))
	}
}

func (s *Suite[S]) seal() {
	s.phase = suiteSealed
}

func (s *Suite[S]) failfAt(where, format string, args ...any) {
	s.owner.Fatalf("bdd: "+where+": "+format, args...)
}

// runLeaf executes one selected leaf in its own subtest:
//
//	disposed:  Skip
//	active:    builder -> Parallel -> setup -> Suite Before -> Scenario Before
//	           -> Background -> visible operations
//
// Error, Fatal, Skip, or panic in any stage stops the stages after it. A
// scenario that stops after its builder has run logs the operation it stopped
// in and the operations that did not run.
func (p *featurePlan[S]) runLeaf(t *testing.T, leaf *leaf[S]) {
	t.Helper()
	if leaf.meta.hasDisposition() {
		t.Skip(leaf.meta.diagnostic(leaf.breadcrumb))
	}
	t.Log(leaf.breadcrumb)
	spec := newScenarioSpec[S](t)
	func() {
		defer spec.seal()
		leaf.build(spec)
	}()
	if halted(t) {
		return
	}
	if len(spec.steps) == 0 {
		t.Fatalf("bdd: %s: scenario %q must contain at least one visible operation", leaf.source, leaf.name)
	}
	if spec.parallel {
		t.Parallel()
	}

	operations := slices.Concat(leaf.before, spec.before, leaf.background, spec.steps)
	current, done, completed := "Setup", 0, false
	// Cleanup rather than defer: it also runs after Fatal, Skip, and panic,
	// and testing attributes its log line to the Feature call site.
	t.Cleanup(func() {
		t.Helper()
		if !completed {
			t.Log(stopReport(current, operations[done:]))
		}
	})

	t.Logf("Setup: %s", leaf.breadcrumb)
	state := p.setup(t)
	if halted(t) {
		return
	}
	for i, operation := range operations {
		current, done = operation.label()+" declared at "+operation.source, i+1
		if operation.kind == kindBefore {
			t.Logf("Before: %s", leaf.breadcrumb)
		} else {
			t.Logf("%s: %s", operation.kind, operation.text)
		}
		operation.fn(t, state)
		if halted(t) {
			return
		}
	}
	completed = true
}

// halted reports whether Error, Fatal, or Skip has stopped the test.
func halted(t *testing.T) bool {
	return t.Failed() || t.Skipped()
}

func stopReport[S any](current string, notRun []step[S]) string {
	report := "bdd: scenario stopped in " + current
	if len(notRun) == 0 {
		return report
	}
	labels := make([]string, 0, len(notRun))
	for _, operation := range notRun {
		labels = append(labels, operation.label())
	}
	return report + "; not run: " + strings.Join(labels, ", ")
}
