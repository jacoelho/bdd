package bdd

import (
	"fmt"
	"io"
	"path"
	"strings"
	"unicode/utf8"
)

// narrative prints one running scenario to its test's output stream: the
// leaf's title, then each operation as it starts. It writes through
// testing.T.Output, so lines carry no testing source prefix; each line ends
// with the site that declared it instead.
type narrative struct {
	out   io.Writer
	width int // widest line, so the declaration sites align
}

func newNarrative[S any](out io.Writer, title string, operations []step[S]) narrative {
	width := utf8.RuneCountInString(title)
	for _, operation := range operations {
		width = max(width, utf8.RuneCountInString(operation.narrate()))
	}
	return narrative{out: out, width: width}
}

// line prints text followed by its declaration site.
func (n narrative) line(text, source string) {
	fmt.Fprintf(n.out, "%-*s  # %s\n", n.width, text, site(source))
}

// site shortens a declaration source to file:line, as testing prints sources.
func site(source string) string {
	return path.Base(source)
}

// stopped reports the stage a scenario stopped in and the operations after it.
func stopped[S any](n narrative, stage string, notRun []step[S]) {
	report := "bdd: scenario stopped in " + stage
	if len(notRun) > 0 {
		labels := make([]string, 0, len(notRun))
		for _, operation := range notRun {
			labels = append(labels, operation.label())
		}
		report += "; not run: " + strings.Join(labels, ", ")
	}
	fmt.Fprintln(n.out, report)
}
