package versioning

import (
	"fmt"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apricote/releaser-pleaser/internal/commitparser"
	"github.com/apricote/releaser-pleaser/internal/git"
)

func TestSemVer_NextVersion(t *testing.T) {
	type args struct {
		releases        git.Releases
		versionBump     VersionBump
		nextVersionType NextVersionType
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "simple bump (major)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1"},
					Stable: &git.Tag{Name: "v1.1.1"},
				},
				versionBump:     MajorVersion,
				nextVersionType: NextVersionTypeUndefined,
			},
			want:    "v2.0.0",
			wantErr: assert.NoError,
		},
		{
			name: "simple bump (minor)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1"},
					Stable: &git.Tag{Name: "v1.1.1"},
				},
				versionBump:     MinorVersion,
				nextVersionType: NextVersionTypeUndefined,
			},
			want:    "v1.2.0",
			wantErr: assert.NoError,
		},
		{
			name: "simple bump (patch)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1"},
					Stable: &git.Tag{Name: "v1.1.1"},
				},
				versionBump:     PatchVersion,
				nextVersionType: NextVersionTypeUndefined,
			},
			want:    "v1.1.2",
			wantErr: assert.NoError,
		},
		{
			name: "normal to prerelease  (major)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1"},
					Stable: &git.Tag{Name: "v1.1.1"},
				},
				versionBump:     MajorVersion,
				nextVersionType: NextVersionTypeRC,
			},
			want:    "v2.0.0-rc.0",
			wantErr: assert.NoError,
		},
		{
			name: "normal to prerelease  (minor)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1"},
					Stable: &git.Tag{Name: "v1.1.1"},
				},
				versionBump:     MinorVersion,
				nextVersionType: NextVersionTypeRC,
			},
			want:    "v1.2.0-rc.0",
			wantErr: assert.NoError,
		},
		{
			name: "normal to prerelease  (patch)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1"},
					Stable: &git.Tag{Name: "v1.1.1"},
				},
				versionBump:     PatchVersion,
				nextVersionType: NextVersionTypeRC,
			},
			want:    "v1.1.2-rc.0",
			wantErr: assert.NoError,
		},
		{
			name: "prerelease bump (major)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v2.0.0-rc.0"},
					Stable: &git.Tag{Name: "v1.1.1"},
				},
				versionBump:     MajorVersion,
				nextVersionType: NextVersionTypeRC,
			},
			want:    "v2.0.0-rc.1",
			wantErr: assert.NoError,
		},
		{
			name: "prerelease bump (minor)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.2.0-rc.0"},
					Stable: &git.Tag{Name: "v1.1.1"},
				},
				versionBump:     MinorVersion,
				nextVersionType: NextVersionTypeRC,
			},
			want:    "v1.2.0-rc.1",
			wantErr: assert.NoError,
		},
		{
			name: "prerelease bump (patch)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.2-rc.0"},
					Stable: &git.Tag{Name: "v1.1.1"},
				},
				versionBump:     PatchVersion,
				nextVersionType: NextVersionTypeRC,
			},
			want:    "v1.1.2-rc.1",
			wantErr: assert.NoError,
		},
		{
			name: "prerelease different bump (major)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.2.0-rc.0"},
					Stable: &git.Tag{Name: "v1.1.1"},
				},
				versionBump:     MajorVersion,
				nextVersionType: NextVersionTypeRC,
			},
			want:    "v2.0.0-rc.1",
			wantErr: assert.NoError,
		},
		{
			name: "prerelease different bump (minor)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.2-rc.0"},
					Stable: &git.Tag{Name: "v1.1.1"},
				},
				versionBump:     MinorVersion,
				nextVersionType: NextVersionTypeRC,
			},
			want:    "v1.2.0-rc.1",
			wantErr: assert.NoError,
		},
		{
			name: "prerelease to prerelease",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1-alpha.2"},
					Stable: &git.Tag{Name: "v1.1.0"},
				},
				versionBump:     PatchVersion,
				nextVersionType: NextVersionTypeRC,
			},
			want:    "v1.1.1-rc.0",
			wantErr: assert.NoError,
		},
		{
			name: "prerelease to normal (explicit)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1-alpha.2"},
					Stable: &git.Tag{Name: "v1.1.0"},
				},
				versionBump:     PatchVersion,
				nextVersionType: NextVersionTypeNormal,
			},
			want:    "v1.1.1",
			wantErr: assert.NoError,
		},
		{
			name: "prerelease to normal (implicit)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1-alpha.2"},
					Stable: &git.Tag{Name: "v1.1.0"},
				},
				versionBump:     PatchVersion,
				nextVersionType: NextVersionTypeUndefined,
			},
			want:    "v1.1.1",
			wantErr: assert.NoError,
		},
		{
			name: "nil tag (major)",
			args: args{
				releases: git.Releases{
					Latest: nil,
					Stable: nil,
				},
				versionBump:     MajorVersion,
				nextVersionType: NextVersionTypeUndefined,
			},
			want:    "v1.0.0",
			wantErr: assert.NoError,
		},
		{
			name: "nil tag (minor)",
			args: args{
				releases: git.Releases{
					Latest: nil,
					Stable: nil,
				},
				versionBump:     MinorVersion,
				nextVersionType: NextVersionTypeUndefined,
			},
			want:    "v0.1.0",
			wantErr: assert.NoError,
		},
		{
			name: "nil tag (patch)",
			args: args{
				releases: git.Releases{
					Latest: nil,
					Stable: nil,
				},
				versionBump:     PatchVersion,
				nextVersionType: NextVersionTypeUndefined,
			},
			want:    "v0.0.1",
			wantErr: assert.NoError,
		},
		{
			name: "nil stable release (major)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1-rc.0"},
					Stable: nil,
				},
				versionBump:     MajorVersion,
				nextVersionType: NextVersionTypeUndefined,
			},
			want:    "v2.0.0",
			wantErr: assert.NoError,
		},
		{
			name: "nil stable release (minor)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1-rc.0"},
					Stable: nil,
				},
				versionBump:     MinorVersion,
				nextVersionType: NextVersionTypeUndefined,
			},
			want:    "v1.2.0",
			wantErr: assert.NoError,
		},
		{
			name: "nil stable release (patch)",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1-rc.0"},
					Stable: nil,
				},
				versionBump:     PatchVersion,
				nextVersionType: NextVersionTypeUndefined,
			},
			// TODO: Is this actually correct our should it be v1.1.1?
			want:    "v1.1.2",
			wantErr: assert.NoError,
		},
		{
			name: "error on invalid tag semver",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "foodazzle"},
					Stable: &git.Tag{Name: "foodazzle"},
				},
				versionBump:     PatchVersion,
				nextVersionType: NextVersionTypeRC,
			},
			want:    "",
			wantErr: assert.Error,
		},
		{
			name: "error on invalid tag prerelease",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1-rc.foo"},
					Stable: &git.Tag{Name: "v1.1.1-rc.foo"},
				},
				versionBump:     PatchVersion,
				nextVersionType: NextVersionTypeRC,
			},
			want:    "",
			wantErr: assert.Error,
		},
		{
			name: "error on invalid bump",
			args: args{
				releases: git.Releases{
					Latest: &git.Tag{Name: "v1.1.1"},
					Stable: &git.Tag{Name: "v1.1.1"},
				},

				versionBump:     UnknownVersion,
				nextVersionType: NextVersionTypeUndefined,
			},
			want:    "",
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SemVer.NextVersion(tt.args.releases, tt.args.versionBump, tt.args.nextVersionType)
			if !tt.wantErr(t, err, fmt.Sprintf("SemVerNextVersion(Releases(%v, %v), %v, %v)", tt.args.releases.Latest, tt.args.releases.Stable, tt.args.versionBump, tt.args.nextVersionType)) {
				return
			}
			assert.Equalf(t, tt.want, got, "SemVerNextVersion(Releases(%v, %v), %v, %v)", tt.args.releases.Latest, tt.args.releases.Stable, tt.args.versionBump, tt.args.nextVersionType)
		})
	}
}

func TestVersionBumpFromCommits(t *testing.T) {
	tests := []struct {
		name            string
		analyzedCommits []commitparser.AnalyzedCommit
		want            VersionBump
	}{
		{
			name:            "no entries (unknown)",
			analyzedCommits: []commitparser.AnalyzedCommit{},
			want:            UnknownVersion,
		},
		{
			name:            "non-release type (unknown)",
			analyzedCommits: []commitparser.AnalyzedCommit{{Type: "docs"}},
			want:            UnknownVersion,
		},
		{
			name:            "single breaking (major)",
			analyzedCommits: []commitparser.AnalyzedCommit{{BreakingChange: true}},
			want:            MajorVersion,
		},
		{
			name:            "single feat (minor)",
			analyzedCommits: []commitparser.AnalyzedCommit{{Type: "feat"}},
			want:            MinorVersion,
		},
		{
			name:            "single fix (patch)",
			analyzedCommits: []commitparser.AnalyzedCommit{{Type: "fix"}},
			want:            PatchVersion,
		},
		{
			name:            "multiple entries (major)",
			analyzedCommits: []commitparser.AnalyzedCommit{{Type: "fix"}, {BreakingChange: true}},
			want:            MajorVersion,
		},
		{
			name:            "multiple entries (minor)",
			analyzedCommits: []commitparser.AnalyzedCommit{{Type: "fix"}, {Type: "feat"}},
			want:            MinorVersion,
		},
		{
			name:            "multiple entries (patch)",
			analyzedCommits: []commitparser.AnalyzedCommit{{Type: "docs"}, {Type: "fix"}},
			want:            PatchVersion,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, BumpFromCommits(tt.analyzedCommits), "BumpFromCommits(%v)", tt.analyzedCommits)
		})
	}
}

func TestSemVer_IsPrerelease(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "empty string",
			version: "",
			want:    false,
		},
		{
			name:    "stable version",
			version: "v1.0.0",
			want:    false,
		},
		{
			name:    "pre-release version",
			version: "v1.0.0-rc.1+foo",
			want:    true,
		},
		{
			name:    "invalid version",
			version: "ajfkdafjdsfj",
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, SemVer.IsPrerelease(tt.version), "IsSemverPrerelease(%v)", tt.version)
		})
	}
}

func TestSemVerWithPrereleaseID_NextVersion(t *testing.T) {
	tests := []struct {
		name            string
		releases        git.Releases
		versionBump     VersionBump
		nextVersionType NextVersionType
		want            string
		wantErr         assert.ErrorAssertionFunc
	}{
		{
			name: "first prerelease after stable",
			releases: git.Releases{
				Latest: &git.Tag{Name: "v1.2.0"},
				Stable: &git.Tag{Name: "v1.2.0"},
			},
			versionBump: MinorVersion,
			want:        "v1.3.0-staging.0",
			wantErr:     assert.NoError,
		},
		{
			name: "next prerelease",
			releases: git.Releases{
				Latest: &git.Tag{Name: "v1.3.0-staging.0"},
				Stable: &git.Tag{Name: "v1.2.0"},
			},
			versionBump: MinorVersion,
			want:        "v1.3.0-staging.1",
			wantErr:     assert.NoError,
		},
		{
			name: "ignores next version type",
			releases: git.Releases{
				Latest: &git.Tag{Name: "v1.3.0-staging.1"},
				Stable: &git.Tag{Name: "v1.2.0"},
			},
			versionBump:     MinorVersion,
			nextVersionType: NextVersionTypeNormal,
			want:            "v1.3.0-staging.2",
			wantErr:         assert.NoError,
		},
		{
			name: "counter restarts when the version changes",
			releases: git.Releases{
				Latest: &git.Tag{Name: "v1.3.0-staging.1"},
				Stable: &git.Tag{Name: "v1.2.0"},
			},
			versionBump: MajorVersion,
			want:        "v2.0.0-staging.0",
			wantErr:     assert.NoError,
		},
		{
			name: "only previous prerelease",
			releases: git.Releases{
				Latest: &git.Tag{Name: "v0.1.0-staging.0"},
			},
			versionBump: PatchVersion,
			want:        "v0.1.1-staging.0",
			wantErr:     assert.NoError,
		},
		{
			name:        "no previous releases",
			releases:    git.Releases{},
			versionBump: PatchVersion,
			want:        "v0.0.1-staging.0",
			wantErr:     assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SemVerWithPrereleaseID("staging").NextVersion(tt.releases, tt.versionBump, tt.nextVersionType)
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIncludesTag(t *testing.T) {
	tests := []struct {
		name         string
		version      string
		prereleaseID string
		want         bool
	}{
		{name: "stable without id", version: "1.2.0", want: true},
		{name: "rc without id", version: "1.2.0-rc.0", want: true},
		{name: "beta without id", version: "1.2.0-beta.0", want: true},
		{name: "alpha without id", version: "1.2.0-alpha.0", want: true},
		{name: "other prerelease without id", version: "1.2.0-staging.0", want: false},
		{name: "stable with id", version: "1.2.0", prereleaseID: "staging", want: true},
		{name: "same prerelease id", version: "1.2.0-staging.0", prereleaseID: "staging", want: true},
		{name: "rc with id", version: "1.2.0-rc.0", prereleaseID: "staging", want: false},
		{name: "other prerelease id", version: "1.2.0-qa.0", prereleaseID: "staging", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := semver.Parse(tt.version)
			require.NoError(t, err)

			assert.Equal(t, tt.want, IncludesTag(version, tt.prereleaseID))
		})
	}
}

func TestValidatePrereleaseID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr assert.ErrorAssertionFunc
	}{
		{id: "", wantErr: assert.NoError},
		{id: "staging", wantErr: assert.NoError},
		{id: "pre-prod", wantErr: assert.NoError},
		{id: "rc", wantErr: assert.Error},
		{id: "beta", wantErr: assert.Error},
		{id: "alpha", wantErr: assert.Error},
		{id: "123", wantErr: assert.Error},
		{id: "staging.1", wantErr: assert.Error},
		{id: "stag_ing", wantErr: assert.Error},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			tt.wantErr(t, ValidatePrereleaseID(tt.id))
		})
	}
}
