package rp

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apricote/releaser-pleaser/internal/commitparser/conventionalcommits"
	"github.com/apricote/releaser-pleaser/internal/forge"
	"github.com/apricote/releaser-pleaser/internal/git"
	"github.com/apricote/releaser-pleaser/internal/releasepr"
	"github.com/apricote/releaser-pleaser/internal/versioning"
)

// fakeForge implements the parts of [forge.Forge] that runReconcileReleasePR uses before it clones the repository.
// Calling any other method panics.
type fakeForge struct {
	forge.Forge

	releases     git.Releases
	commitsSince map[string][]git.Commit
	pr           *releasepr.ReleasePullRequest
	closed       []*releasepr.ReleasePullRequest
}

func (f *fakeForge) PullRequestForBranch(context.Context, string) (*releasepr.ReleasePullRequest, error) {
	return f.pr, nil
}

func (f *fakeForge) LatestTags(context.Context) (git.Releases, error) {
	return f.releases, nil
}

func (f *fakeForge) CommitsSince(_ context.Context, tag *git.Tag) ([]git.Commit, error) {
	name := ""
	if tag != nil {
		name = tag.Name
	}
	return f.commitsSince[name], nil
}

func (f *fakeForge) ClosePullRequest(_ context.Context, pr *releasepr.ReleasePullRequest) error {
	f.closed = append(f.closed, pr)
	return nil
}

func TestReconcileReleasePR_NoChangesSinceLatestPrerelease(t *testing.T) {
	stable := &git.Tag{Hash: "aaa", Name: "v1.2.0"}
	staging := &git.Tag{Hash: "bbb", Name: "v1.3.0-staging.0"}
	feat := git.Commit{Hash: "ccc", Message: "feat: new feature"}
	release := git.Commit{Hash: "ddd", Message: "chore(staging): release v1.3.0-staging.0"}

	newRP := func(f *fakeForge) *ReleaserPleaser {
		logger := slog.New(slog.DiscardHandler)
		return New(f, logger, "staging", conventionalcommits.NewParser(logger), versioning.SemVerWithPrereleaseID("staging"), nil, nil)
	}

	t.Run("no pull request is opened", func(t *testing.T) {
		f := &fakeForge{
			releases: git.Releases{Latest: staging, Stable: stable},
			commitsSince: map[string][]git.Commit{
				stable.Name:  {feat, release},
				staging.Name: {},
			},
		}

		require.NoError(t, newRP(f).runReconcileReleasePR(t.Context()))
		assert.Empty(t, f.closed)
	})

	t.Run("existing pull request is closed", func(t *testing.T) {
		pr := &releasepr.ReleasePullRequest{PullRequest: git.PullRequest{ID: 1, Title: "chore(staging): release v1.3.0-staging.1"}}
		f := &fakeForge{
			releases: git.Releases{Latest: staging, Stable: stable},
			commitsSince: map[string][]git.Commit{
				stable.Name:  {feat, release},
				staging.Name: {release},
			},
			pr: pr,
		}

		require.NoError(t, newRP(f).runReconcileReleasePR(t.Context()))
		assert.Equal(t, []*releasepr.ReleasePullRequest{pr}, f.closed)
	})
}
