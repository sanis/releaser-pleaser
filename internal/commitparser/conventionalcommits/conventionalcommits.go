package conventionalcommits

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/leodido/go-conventionalcommits"
	"github.com/leodido/go-conventionalcommits/parser"

	"github.com/apricote/releaser-pleaser/internal/commitparser"
	"github.com/apricote/releaser-pleaser/internal/git"
)

// whitespaceRegex matches any run of whitespace, used to normalize commit subjects before
// comparing them for duplicates.
var whitespaceRegex = regexp.MustCompile(`\s+`)

type Parser struct {
	machine conventionalcommits.Machine
	logger  *slog.Logger
}

func NewParser(logger *slog.Logger) *Parser {
	parserMachine := parser.NewMachine(
		parser.WithBestEffort(),
		parser.WithTypes(conventionalcommits.TypesConventional),
	)

	return &Parser{
		machine: parserMachine,
		logger:  logger,
	}
}

// Analyze turns the commits into changelog entries. Every commit is kept, commits that are not
// valid conventional commits are reported as commitparser.TypeOther. Merge commits and commits
// that repeat a subject seen earlier in the same release window are skipped.
func (c *Parser) Analyze(commits []git.Commit) ([]commitparser.AnalyzedCommit, error) {
	analyzedCommits := make([]commitparser.AnalyzedCommit, 0, len(commits))
	seenSubjects := make(map[string]struct{}, len(commits))

	for _, commit := range commits {
		if commit.IsMerge() {
			c.logger.Debug("commit is a merge commit, skipping", "commit.hash", commit.Hash)
			continue
		}

		subject := commit.Subject()

		normalizedSubject := normalizeSubject(subject)
		if _, seen := seenSubjects[normalizedSubject]; seen {
			c.logger.Debug("commit repeats an earlier subject, skipping", "commit.hash", commit.Hash, "commit.subject", subject)
			continue
		}
		seenSubjects[normalizedSubject] = struct{}{}

		analyzedCommit, err := c.analyzeCommit(commit, subject)
		if err != nil {
			return nil, err
		}

		analyzedCommits = append(analyzedCommits, analyzedCommit)
	}

	return analyzedCommits, nil
}

func (c *Parser) analyzeCommit(commit git.Commit, subject string) (commitparser.AnalyzedCommit, error) {
	msg, err := c.machine.Parse([]byte(strings.TrimSpace(commit.Message)))
	if err != nil {
		if msg == nil {
			c.logger.Debug("failed to parse message of commit, using it as-is", "commit.hash", commit.Hash, "err", err)
			return otherCommit(commit, subject), nil
		}

		c.logger.Debug("failed to parse message of commit fully, trying to use as much as possible", "commit.hash", commit.Hash, "err", err)
	}

	conventionalCommit, ok := msg.(*conventionalcommits.ConventionalCommit)
	if !ok {
		return commitparser.AnalyzedCommit{}, fmt.Errorf("unable to get ConventionalCommit from parser result: %T", msg)
	}

	if conventionalCommit.Type == "" {
		// Parsing broke before getting the type, we can only use the commit as-is.
		c.logger.Debug("commit type was not parsed, using commit as-is", "commit.hash", commit.Hash, "err", err)
		return otherCommit(commit, subject), nil
	}

	description := conventionalCommit.Description
	if description == "" {
		// Parsing broke before getting the description. The subject is a better changelog entry
		// than an empty line.
		description = subject
	}

	return commitparser.AnalyzedCommit{
		Commit:         commit,
		Type:           conventionalCommit.Type,
		Description:    description,
		Scope:          conventionalCommit.Scope,
		BreakingChange: conventionalCommit.IsBreakingChange(),
	}, nil
}

// otherCommit describes a commit that is not a conventional commit. Its subject is the whole
// information we have, and there is no scope or breaking change marker to read.
func otherCommit(commit git.Commit, subject string) commitparser.AnalyzedCommit {
	return commitparser.AnalyzedCommit{
		Commit:      commit,
		Type:        commitparser.TypeOther,
		Description: subject,
	}
}

// normalizeSubject lowercases the subject and collapses runs of whitespace, so that commits which
// only differ in capitalization or spacing count as duplicates of each other.
func normalizeSubject(subject string) string {
	return whitespaceRegex.ReplaceAllString(strings.ToLower(strings.TrimSpace(subject)), " ")
}
