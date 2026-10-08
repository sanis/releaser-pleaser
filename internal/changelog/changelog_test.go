package changelog

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apricote/releaser-pleaser/internal/commitparser"
	"github.com/apricote/releaser-pleaser/internal/git"
	"github.com/apricote/releaser-pleaser/internal/testdata"
)

func ptr[T any](input T) *T {
	return &input
}

func Test_NewChangelogEntry(t *testing.T) {
	type args struct {
		analyzedCommits []commitparser.AnalyzedCommit
		version         string
		link            string
		compare         string
		prefix          string
		suffix          string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "empty",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{},
				version:         "1.0.0",
				link:            "https://example.com/1.0.0",
			},
			want:    "## [1.0.0](https://example.com/1.0.0)\n",
			wantErr: assert.NoError,
		},
		{
			name: "single feature",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{
					{
						Commit:      git.Commit{Hash: "abc1234567890", URL: "https://example.com/commit/abc1234567890"},
						Type:        "feat",
						Description: "Foobar!",
					},
				},
				version: "1.0.0",
				link:    "https://example.com/1.0.0",
			},
			want:    "## [1.0.0](https://example.com/1.0.0)\n\n### Features\n\n- Foobar! ([abc1234](https://example.com/commit/abc1234567890))\n",
			wantErr: assert.NoError,
		},
		{
			name: "single breaking change",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{
					{
						Commit:         git.Commit{Hash: "abc1234567890", URL: "https://example.com/commit/abc1234567890"},
						Type:           "feat",
						Description:    "Foobar!",
						BreakingChange: true,
					},
				},
				version: "1.0.0",
				link:    "https://example.com/1.0.0",
			},
			want:    "## [1.0.0](https://example.com/1.0.0)\n\n### Features\n\n- **BREAKING**: Foobar! ([abc1234](https://example.com/commit/abc1234567890))\n",
			wantErr: assert.NoError,
		},
		{
			name: "single fix",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{
					{
						Commit:      git.Commit{Hash: "abc1234567890", URL: "https://example.com/commit/abc1234567890"},
						Type:        "fix",
						Description: "Foobar!",
					},
				},
				version: "1.0.0",
				link:    "https://example.com/1.0.0",
			},
			want:    "## [1.0.0](https://example.com/1.0.0)\n\n### Bug Fixes\n\n- Foobar! ([abc1234](https://example.com/commit/abc1234567890))\n",
			wantErr: assert.NoError,
		},
		{
			name: "multiple commits with scopes",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{
					{
						Commit:      git.Commit{Hash: "aaa1111111111", URL: "https://example.com/commit/aaa1111111111"},
						Type:        "feat",
						Description: "Blabla!",
					},
					{
						Commit:      git.Commit{Hash: "bbb2222222222", URL: "https://example.com/commit/bbb2222222222"},
						Type:        "feat",
						Description: "So awesome!",
						Scope:       ptr("awesome"),
					},
					{
						Commit:      git.Commit{Hash: "ccc3333333333", URL: "https://example.com/commit/ccc3333333333"},
						Type:        "fix",
						Description: "Foobar!",
					},
					{
						Commit:      git.Commit{Hash: "ddd4444444444", URL: "https://example.com/commit/ddd4444444444"},
						Type:        "fix",
						Description: "So sad!",
						Scope:       ptr("sad"),
					},
				},
				version: "1.0.0",
				link:    "https://example.com/1.0.0",
			},
			want: `## [1.0.0](https://example.com/1.0.0)

### Features

- Blabla! ([aaa1111](https://example.com/commit/aaa1111111111))
- **awesome**: So awesome! ([bbb2222](https://example.com/commit/bbb2222222222))

### Bug Fixes

- Foobar! ([ccc3333](https://example.com/commit/ccc3333333333))
- **sad**: So sad! ([ddd4444](https://example.com/commit/ddd4444444444))
`,
			wantErr: assert.NoError,
		},
		{
			name: "non-conventional commit renders as other",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{
					{
						Commit:      git.Commit{Hash: "abc1234567890", URL: "https://example.com/commit/abc1234567890"},
						Type:        commitparser.TypeOther,
						Description: "Update the readme",
					},
				},
				version: "1.0.0",
				link:    "https://example.com/1.0.0",
			},
			want:    "## [1.0.0](https://example.com/1.0.0)\n\n### Other Changes\n\n- Update the readme ([abc1234](https://example.com/commit/abc1234567890))\n",
			wantErr: assert.NoError,
		},
		{
			name: "neutral conventional type renders with its own section",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{
					{
						Commit:      git.Commit{Hash: "abc1234567890", URL: "https://example.com/commit/abc1234567890"},
						Type:        "chore",
						Description: "Bump dependencies",
					},
				},
				version: "1.0.0",
				link:    "https://example.com/1.0.0",
			},
			want:    "## [1.0.0](https://example.com/1.0.0)\n\n### Chores\n\n- Bump dependencies ([abc1234](https://example.com/commit/abc1234567890))\n",
			wantErr: assert.NoError,
		},
		{
			name: "unmapped type falls back to the raw type name",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{
					{
						Commit:      git.Commit{Hash: "abc1234567890", URL: "https://example.com/commit/abc1234567890"},
						Type:        "wibble",
						Description: "Something new under the sun",
					},
				},
				version: "1.0.0",
				link:    "https://example.com/1.0.0",
			},
			want:    "## [1.0.0](https://example.com/1.0.0)\n\n### wibble\n\n- Something new under the sun ([abc1234](https://example.com/commit/abc1234567890))\n",
			wantErr: assert.NoError,
		},
		{
			name: "section order: features, fixes, alphabetical, other last",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{
					{
						Commit:      git.Commit{Hash: "fff6666666666", URL: "https://example.com/commit/fff6666666666"},
						Type:        commitparser.TypeOther,
						Description: "Just a commit",
					},
					{
						Commit:      git.Commit{Hash: "eee5555555555", URL: "https://example.com/commit/eee5555555555"},
						Type:        "wibble",
						Description: "Unknown type!",
					},
					{
						Commit:      git.Commit{Hash: "ddd4444444444", URL: "https://example.com/commit/ddd4444444444"},
						Type:        "ci",
						Description: "CI!",
					},
					{
						Commit:      git.Commit{Hash: "ccc3333333333", URL: "https://example.com/commit/ccc3333333333"},
						Type:        "chore",
						Description: "Chore!",
					},
					{
						Commit:      git.Commit{Hash: "bbb2222222222", URL: "https://example.com/commit/bbb2222222222"},
						Type:        "fix",
						Description: "Fix!",
					},
					{
						Commit:      git.Commit{Hash: "aaa1111111111", URL: "https://example.com/commit/aaa1111111111"},
						Type:        "feat",
						Description: "Feature!",
					},
				},
				version: "1.0.0",
				link:    "https://example.com/1.0.0",
			},
			want: `## [1.0.0](https://example.com/1.0.0)

### Features

- Feature! ([aaa1111](https://example.com/commit/aaa1111111111))

### Bug Fixes

- Fix! ([bbb2222](https://example.com/commit/bbb2222222222))

### Chores

- Chore! ([ccc3333](https://example.com/commit/ccc3333333333))

### Continuous Integration

- CI! ([ddd4444](https://example.com/commit/ddd4444444444))

### wibble

- Unknown type! ([eee5555](https://example.com/commit/eee5555555555))

### Other Changes

- Just a commit ([fff6666](https://example.com/commit/fff6666666666))
`,
			wantErr: assert.NoError,
		},
		{
			name: "bare version without the v prefix",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{
					{
						Commit:      git.Commit{Hash: "abc1234567890", URL: "https://example.com/commit/abc1234567890"},
						Type:        "fix",
						Description: "Foobar!",
					},
				},
				version: "1.396.0",
				link:    "https://example.com/1.396.0",
				compare: "https://example.com/compare/1.395.0/1.396.0/",
			},
			want: `## [1.396.0](https://example.com/1.396.0)

[Compare to previous version](https://example.com/compare/1.395.0/1.396.0/)

### Bug Fixes

- Foobar! ([abc1234](https://example.com/commit/abc1234567890))
`,
			wantErr: assert.NoError,
		},
		{
			name: "compare url",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{
					{
						Commit:      git.Commit{Hash: "abc1234567890", URL: "https://example.com/commit/abc1234567890"},
						Type:        "fix",
						Description: "Foobar!",
					},
				},
				version: "1.0.0",
				link:    "https://example.com/1.0.0",
				compare: "https://example.com/compare/0.1.0/1.0.0/",
			},
			want:    testdata.MustReadFileString(t, "changelog-entry-compare-url.txt"),
			wantErr: assert.NoError,
		},
		{
			name: "prefix",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{
					{
						Commit:      git.Commit{Hash: "abc1234567890", URL: "https://example.com/commit/abc1234567890"},
						Type:        "fix",
						Description: "Foobar!",
					},
				},
				version: "1.0.0",
				link:    "https://example.com/1.0.0",
				prefix:  testdata.MustReadFileString(t, "prefix.txt"),
			},
			want:    testdata.MustReadFileString(t, "changelog-entry-prefix.txt"),
			wantErr: assert.NoError,
		},
		{
			name: "suffix",
			args: args{
				analyzedCommits: []commitparser.AnalyzedCommit{
					{
						Commit:      git.Commit{Hash: "abc1234567890", URL: "https://example.com/commit/abc1234567890"},
						Type:        "fix",
						Description: "Foobar!",
					},
				},
				version: "1.0.0",
				link:    "https://example.com/1.0.0",
				suffix:  testdata.MustReadFileString(t, "suffix.txt"),
			},
			want:    testdata.MustReadFileString(t, "changelog-entry-suffix.txt"),
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := New(commitparser.ByType(tt.args.analyzedCommits), tt.args.version, tt.args.link, tt.args.compare, tt.args.prefix, tt.args.suffix)
			got, err := Entry(slog.Default(), DefaultTemplate(), data, Formatting{})
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
