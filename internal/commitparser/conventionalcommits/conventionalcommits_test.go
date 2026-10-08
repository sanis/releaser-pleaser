package conventionalcommits

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apricote/releaser-pleaser/internal/commitparser"
	"github.com/apricote/releaser-pleaser/internal/git"
)

func TestAnalyzeCommits(t *testing.T) {
	tests := []struct {
		name            string
		commits         []git.Commit
		expectedCommits []commitparser.AnalyzedCommit
		wantErr         assert.ErrorAssertionFunc
	}{
		{
			name:            "empty commits",
			commits:         []git.Commit{},
			expectedCommits: []commitparser.AnalyzedCommit{},
			wantErr:         assert.NoError,
		},
		{
			name: "keeps malformed commit message as other",
			commits: []git.Commit{
				{
					Message: "aksdjaklsdjka",
				},
			},
			expectedCommits: []commitparser.AnalyzedCommit{
				{
					Commit:      git.Commit{Message: "aksdjaklsdjka"},
					Type:        commitparser.TypeOther,
					Description: "aksdjaklsdjka",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "other uses the subject as description",
			commits: []git.Commit{
				{
					Message: "  Update the readme  \n\nWith a body that is not part of the subject.\n",
				},
			},
			expectedCommits: []commitparser.AnalyzedCommit{
				{
					Commit:         git.Commit{Message: "  Update the readme  \n\nWith a body that is not part of the subject.\n"},
					Type:           commitparser.TypeOther,
					Description:    "Update the readme",
					Scope:          nil,
					BreakingChange: false,
				},
			},
			wantErr: assert.NoError,
		},
		{
			// GitLab seems to create commits with pattern "scope: message\n" if no body is added.
			// This has previously caused a parser error "missing a blank line".
			// We added a workaround with `strings.TrimSpace()` and this test make sure that it does not break again.
			name: "handles title with new line",
			commits: []git.Commit{
				{
					Message: "fix: blabla\n",
				},
			},
			expectedCommits: []commitparser.AnalyzedCommit{
				{
					Commit:      git.Commit{Message: "fix: blabla\n"},
					Type:        "fix",
					Description: "blabla",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "keeps neutral conventional types",
			commits: []git.Commit{
				{
					Message: "chore: foobar",
				},
			},
			expectedCommits: []commitparser.AnalyzedCommit{
				{
					Commit:      git.Commit{Message: "chore: foobar"},
					Type:        "chore",
					Description: "foobar",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "skips merge commit (parents)",
			commits: []git.Commit{
				{
					Message: "feat: something useful",
					Parents: []string{"aaa"},
				},
				{
					Message: "chore: this merge has a custom message",
					Parents: []string{"aaa", "bbb"},
				},
			},
			expectedCommits: []commitparser.AnalyzedCommit{
				{
					Commit:      git.Commit{Message: "feat: something useful", Parents: []string{"aaa"}},
					Type:        "feat",
					Description: "something useful",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "skips merge commit (message fallback)",
			commits: []git.Commit{
				{
					Message: "Merge pull request #12 from foo/bar\n\nfeat: sneaky",
				},
				{
					Message: "Merge branch 'main' into feature",
				},
				{
					Message: "Merge remote-tracking branch 'origin/main'",
				},
				{
					Message: "fix: blabla",
				},
			},
			expectedCommits: []commitparser.AnalyzedCommit{
				{
					Commit:      git.Commit{Message: "fix: blabla"},
					Type:        "fix",
					Description: "blabla",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "keeps a commit that only looks like a merge",
			commits: []git.Commit{
				{
					Message: "feat: Merge branch handling into the sync loop",
				},
			},
			expectedCommits: []commitparser.AnalyzedCommit{
				{
					Commit:      git.Commit{Message: "feat: Merge branch handling into the sync loop"},
					Type:        "feat",
					Description: "Merge branch handling into the sync loop",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "skips duplicate subjects, keeping the first",
			commits: []git.Commit{
				{
					Hash:    "aaa",
					Message: "fix: blabla",
				},
				{
					Hash:    "bbb",
					Message: "fix:   BlaBla  ",
				},
				{
					Hash:    "ccc",
					Message: "fix: blabla\n\nA different body does not make it a different subject.",
				},
			},
			expectedCommits: []commitparser.AnalyzedCommit{
				{
					Commit:      git.Commit{Hash: "aaa", Message: "fix: blabla"},
					Type:        "fix",
					Description: "blabla",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "highest bump (patch)",
			commits: []git.Commit{
				{
					Message: "chore: foobar",
				},
				{
					Message: "fix: blabla",
				},
			},
			expectedCommits: []commitparser.AnalyzedCommit{
				{
					Commit:      git.Commit{Message: "chore: foobar"},
					Type:        "chore",
					Description: "foobar",
				},
				{
					Commit:      git.Commit{Message: "fix: blabla"},
					Type:        "fix",
					Description: "blabla",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "highest bump (minor)",
			commits: []git.Commit{
				{
					Message: "fix: blabla",
				},
				{
					Message: "feat: foobar",
				},
			},
			expectedCommits: []commitparser.AnalyzedCommit{
				{
					Commit:      git.Commit{Message: "fix: blabla"},
					Type:        "fix",
					Description: "blabla",
				},
				{
					Commit:      git.Commit{Message: "feat: foobar"},
					Type:        "feat",
					Description: "foobar",
				},
			},
			wantErr: assert.NoError,
		},

		{
			name: "highest bump (major)",
			commits: []git.Commit{
				{
					Message: "fix: blabla",
				},
				{
					Message: "feat!: foobar",
				},
			},
			expectedCommits: []commitparser.AnalyzedCommit{
				{
					Commit:      git.Commit{Message: "fix: blabla"},
					Type:        "fix",
					Description: "blabla",
				},
				{
					Commit:         git.Commit{Message: "feat!: foobar"},
					Type:           "feat",
					Description:    "foobar",
					BreakingChange: true,
				},
			},
			wantErr: assert.NoError,
		},

		{
			name: "success with body",
			commits: []git.Commit{
				{
					Message: "feat: some thing (hz/fl!144)\n\nFixes #15\n\nDepends on !143",
				},
			},
			expectedCommits: []commitparser.AnalyzedCommit{
				{
					Commit:         git.Commit{Message: "feat: some thing (hz/fl!144)\n\nFixes #15\n\nDepends on !143"},
					Type:           "feat",
					Description:    "some thing (hz/fl!144)",
					BreakingChange: false,
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzedCommits, err := NewParser(slog.Default()).Analyze(tt.commits)
			if !tt.wantErr(t, err) {
				return
			}

			assert.Equal(t, tt.expectedCommits, analyzedCommits)
		})
	}
}
