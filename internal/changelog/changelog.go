package changelog

import (
	"bytes"
	_ "embed"
	"log"
	"log/slog"
	"slices"
	"strings"
	"text/template"

	"github.com/apricote/releaser-pleaser/internal/commitparser"
	"github.com/apricote/releaser-pleaser/internal/markdown"
)

var (
	changelogTemplate *template.Template
)

//go:embed changelog.md.tpl
var rawChangelogTemplate string

func init() {
	var err error
	changelogTemplate, err = template.New("changelog").Parse(rawChangelogTemplate)
	if err != nil {
		log.Fatalf("failed to parse changelog template: %v", err)
	}
}

func DefaultTemplate() *template.Template {
	return changelogTemplate
}

type Data struct {
	Commits     map[string][]commitparser.AnalyzedCommit
	Version     string
	VersionLink string
	CompareURL  string
	Prefix      string
	Suffix      string
}

// SectionTitles maps the commit types we know about to a human readable section title. Any type
// that is missing here is rendered with the raw type name as its title, so that a type we have
// never seen still shows up in the changelog.
var SectionTitles = map[string]string{
	commitparser.TypeFeature: "Features",
	commitparser.TypeFix:     "Bug Fixes",
	"perf":                   "Performance Improvements",
	"revert":                 "Reverts",
	"docs":                   "Documentation",
	"style":                  "Styles",
	"refactor":               "Code Refactoring",
	"test":                   "Tests",
	"build":                  "Build System",
	"ci":                     "Continuous Integration",
	"chore":                  "Chores",
	commitparser.TypeOther:   "Other Changes",
}

// pinnedTypes are rendered first, in this order. Every other type follows in alphabetical order,
// with commitparser.TypeOther always coming last.
var pinnedTypes = []string{commitparser.TypeFeature, commitparser.TypeFix}

// Section is one group of commits in the changelog.
type Section struct {
	Type    string
	Title   string
	Commits []commitparser.AnalyzedCommit
}

// Sections returns the non-empty commit groups in the order they should appear in the changelog.
func (d Data) Sections() []Section {
	types := make([]string, 0, len(d.Commits))
	for commitType := range d.Commits {
		if len(d.Commits[commitType]) == 0 {
			continue
		}

		types = append(types, commitType)
	}

	slices.SortFunc(types, func(a, b string) int {
		if rank := typeRank(a) - typeRank(b); rank != 0 {
			return rank
		}

		return strings.Compare(a, b)
	})

	sections := make([]Section, 0, len(types))
	for _, commitType := range types {
		sections = append(sections, Section{
			Type:    commitType,
			Title:   SectionTitle(commitType),
			Commits: d.Commits[commitType],
		})
	}

	return sections
}

// SectionTitle returns the human readable title for the commit type, falling back to the type
// name itself.
func SectionTitle(commitType string) string {
	if title, ok := SectionTitles[commitType]; ok {
		return title
	}

	return commitType
}

func typeRank(commitType string) int {
	if index := slices.Index(pinnedTypes, commitType); index >= 0 {
		return index
	}

	if commitType == commitparser.TypeOther {
		return len(pinnedTypes) + 1
	}

	return len(pinnedTypes)
}

func New(commits map[string][]commitparser.AnalyzedCommit, version, versionLink, compareURL, prefix, suffix string) Data {
	return Data{
		Commits:     commits,
		Version:     version,
		VersionLink: versionLink,
		CompareURL:  compareURL,
		Prefix:      prefix,
		Suffix:      suffix,
	}
}

type Formatting struct {
	HideVersionTitle bool
}

func Entry(logger *slog.Logger, tpl *template.Template, data Data, formatting Formatting) (string, error) {
	var changelog bytes.Buffer
	err := tpl.Execute(&changelog, map[string]any{
		"Data":       data,
		"Formatting": formatting,
	})
	if err != nil {
		return "", err
	}

	formatted, err := markdown.Format(changelog.String())
	if err != nil {
		logger.Warn("failed to format changelog entry, using unformatted", "error", err)
		return changelog.String(), nil
	}

	return formatted, nil
}
