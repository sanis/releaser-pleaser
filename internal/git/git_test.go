package git

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthor_signature(t *testing.T) {
	now := time.Now()

	tests := []struct {
		author Author
		want   *object.Signature
	}{
		{author: Author{Name: "foo", Email: "bar@example.com"}, want: &object.Signature{Name: "foo", Email: "bar@example.com", When: now}},
		{author: Author{Name: "bar", Email: "foo@example.com"}, want: &object.Signature{Name: "bar", Email: "foo@example.com", When: now}},
	}
	for i, tt := range tests {
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			if got := tt.author.signature(now); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("signature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthor_String(t *testing.T) {
	tests := []struct {
		author Author
		want   string
	}{
		{author: Author{Name: "foo", Email: "bar@example.com"}, want: "foo <bar@example.com>"},
		{author: Author{Name: "bar", Email: "foo@example.com"}, want: "bar <foo@example.com>"},
	}
	for i, tt := range tests {
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			if got := tt.author.String(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

const testMainBranch = "main"
const testPRBranch = "releaser-pleaser"

func TestRepository_HasChangesWithRemote(t *testing.T) {
	// go-git/v5 has a bug where it tries to delete the repo root dir (".") multiple times if there is no file left in it.
	// this happens while switching branches in worktree.go rmFileAndDirsIfEmpty.
	// TODO: Fix bug upstream
	// For now I just make sure that there is always at least one file left in the dir by adding an empty "README.md" in the test util.

	mainBranchRef := plumbing.NewBranchReferenceName(testMainBranch)
	localPRBranchRef := plumbing.NewBranchReferenceName(testPRBranch)
	remotePRBranchRef := plumbing.NewBranchReferenceName("remote/" + testPRBranch)

	tests := []struct {
		name    string
		repo    TestRepo
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "no remote pr branch",
			repo: WithTestRepo(
				WithCommit(
					"chore: release v1.0.0",
					WithFile("VERSION", "v1.0.0"),
				),
				WithCommit(
					"chore: release v1.1.0",
					OnBranch(mainBranchRef),
					AsNewBranch(localPRBranchRef),
					WithFile("VERSION", "v1.1.0"),
				),
			),
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "remote pr branch matches local",
			repo: WithTestRepo(
				WithCommit(
					"chore: release v1.0.0",
					WithFile("VERSION", "v1.0.0"),
				),
				WithCommit(
					"chore: release v1.1.0",
					OnBranch(mainBranchRef),
					AsNewBranch(remotePRBranchRef),
					WithFile("VERSION", "v1.1.0"),
				),
				WithCommit(
					"chore: release v1.1.0",
					OnBranch(mainBranchRef),
					AsNewBranch(localPRBranchRef),
					WithFile("VERSION", "v1.1.0"),
				),
			),
			want:    false,
			wantErr: assert.NoError,
		},
		{
			name: "remote pr only needs rebase",
			repo: WithTestRepo(
				WithCommit(
					"chore: release v1.0.0",
					WithFile("VERSION", "v1.0.0"),
				),
				WithCommit(
					"chore: release v1.1.0",
					OnBranch(mainBranchRef),
					AsNewBranch(remotePRBranchRef),
					WithFile("VERSION", "v1.1.0"),
				),
				WithCommit(
					"feat: new feature on remote",
					OnBranch(mainBranchRef),
					WithFile("feature", "yes"),
				),
				WithCommit(
					"chore: release v1.1.0",
					OnBranch(mainBranchRef),
					AsNewBranch(localPRBranchRef),
					WithFile("VERSION", "v1.1.0"),
				),
			),
			want:    false,
			wantErr: assert.NoError,
		},
		{
			name: "needs update",
			repo: WithTestRepo(
				WithCommit(
					"chore: release v1.0.0",
					WithFile("VERSION", "v1.0.0"),
				),
				WithCommit(
					"chore: release v1.1.0",
					OnBranch(mainBranchRef),
					AsNewBranch(remotePRBranchRef),
					WithFile("VERSION", "v1.1.0"),
					WithFile("CHANGELOG.md", "Foo"),
				),
				WithCommit(
					"chore: release v1.1.0",
					OnBranch(mainBranchRef),
					AsNewBranch(localPRBranchRef),
					WithFile("VERSION", "v1.1.0"),
					WithFile("CHANGELOG.md", "FooBar"),
				),
			),
			want:    true,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo(t)
			got, err := repo.hasChangesWithRemote(context.Background(), mainBranchRef, localPRBranchRef, remotePRBranchRef)
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRepository_CommitUsesAuthorAsCommitter(t *testing.T) {
	repo := WithTestRepo()(t)
	author := Author{Name: "release bot", Email: "release@example.com"}

	err := repo.UpdateFile(context.Background(), "README.md", false, func(content string) (string, error) {
		return content + "\nrelease", nil
	})
	require.NoError(t, err)

	commit, err := repo.Commit(context.Background(), "chore: release v1.2.3", author)
	require.NoError(t, err)

	obj, err := repo.r.CommitObject(plumbing.NewHash(commit.Hash))
	require.NoError(t, err)

	assert.Equal(t, author.Name, obj.Author.Name)
	assert.Equal(t, author.Email, obj.Author.Email)
	assert.Equal(t, author.Name, obj.Committer.Name)
	assert.Equal(t, author.Email, obj.Committer.Email)
}

func TestCommit_Subject(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "empty",
			message: "",
			want:    "",
		},
		{
			name:    "single line",
			message: "Update the readme",
			want:    "Update the readme",
		},
		{
			name:    "trailing new line",
			message: "fix: blabla\n",
			want:    "fix: blabla",
		},
		{
			name:    "surrounding whitespace",
			message: "  Update the readme  ",
			want:    "Update the readme",
		},
		{
			name:    "message with body",
			message: "feat: add a thing\n\nWith an explanation of the thing.\n",
			want:    "feat: add a thing",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Commit{Message: tt.message}.Subject())
		})
	}
}

func TestCommit_IsMerge(t *testing.T) {
	tests := []struct {
		name   string
		commit Commit
		want   bool
	}{
		{
			name:   "single parent",
			commit: Commit{Message: "feat: something", Parents: []string{"aaa"}},
			want:   false,
		},
		{
			name:   "two parents",
			commit: Commit{Message: "chore: a custom merge message", Parents: []string{"aaa", "bbb"}},
			want:   true,
		},
		{
			name:   "octopus merge",
			commit: Commit{Message: "Merge branches", Parents: []string{"aaa", "bbb", "ccc"}},
			want:   true,
		},
		{
			name:   "parents win over the message",
			commit: Commit{Message: "Merge branch 'main' into feature", Parents: []string{"aaa"}},
			want:   false,
		},
		{
			name:   "no parents, merge pull request message",
			commit: Commit{Message: "Merge pull request #12 from foo/bar\n\nfeat: sneaky"},
			want:   true,
		},
		{
			name:   "no parents, merge branch message",
			commit: Commit{Message: "Merge branch 'main' into feature"},
			want:   true,
		},
		{
			name:   "no parents, merge remote-tracking branch message",
			commit: Commit{Message: "Merge remote-tracking branch 'origin/main'"},
			want:   true,
		},
		{
			name:   "no parents, regular message",
			commit: Commit{Message: "feat: Merge branch handling into the sync loop"},
			want:   false,
		},
		{
			name:   "no parents, root commit",
			commit: Commit{Message: "Initial commit"},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.commit.IsMerge())
		})
	}
}
