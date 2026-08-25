package commitparser

import (
	"github.com/apricote/releaser-pleaser/internal/git"
)

const (
	TypeFeature = "feat"
	TypeFix     = "fix"

	// TypeOther is the type reported for commits that are not conventional commits. It has no
	// meaning to the conventional commits spec, we use it so that these commits still show up in
	// the changelog instead of being dropped.
	TypeOther = "other"
)

type CommitParser interface {
	Analyze(commits []git.Commit) ([]AnalyzedCommit, error)
}

type AnalyzedCommit struct {
	git.Commit
	Type           string
	Description    string
	Scope          *string
	BreakingChange bool
}

// ByType groups the Commits by the type field. Used by the Changelog.
func ByType(in []AnalyzedCommit) map[string][]AnalyzedCommit {
	out := map[string][]AnalyzedCommit{}

	for _, commit := range in {
		if out[commit.Type] == nil {
			out[commit.Type] = make([]AnalyzedCommit, 0, 1)
		}

		out[commit.Type] = append(out[commit.Type], commit)
	}

	return out
}
