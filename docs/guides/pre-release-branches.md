# Pre-release branches

Some projects deploy a second branch, for example `staging`, before a change is released from `main`. You can run `releaser-pleaser` for that branch too. It then opens a separate release pull request into the branch and releases every version on it as a [pre-release](pre-releases.md) with an identifier that you choose:

- `main` gets stable releases: `v1.3.0`, `v1.4.0`, …
- `staging` gets pre-releases: `v1.3.0-staging.0`, `v1.3.0-staging.1`, …

## Setup

Run `releaser-pleaser` once for each branch and set `prerelease-id` on the run for the pre-release branch.

### GitLab CI/CD

Include the component a second time. Each include needs its own `job-name`:

```yaml
# .gitlab-ci.yml
include:
  - component: $CI_SERVER_FQDN/apricote/releaser-pleaser/run@<version>
    inputs:
      token: $RELEASER_PLEASER_TOKEN

  - component: $CI_SERVER_FQDN/apricote/releaser-pleaser/run@<version>
    inputs:
      token: $RELEASER_PLEASER_TOKEN
      branch: staging
      prerelease-id: staging
      job-name: releaser-pleaser-staging
```

### GitHub Actions

Run the action for each branch with its own `branch` input:

```yaml
# .github/workflows/releaser-pleaser.yaml
name: releaser-pleaser

on:
  push:
    branches: [main, staging]
  pull_request_target:
    types:
      - edited
      - labeled
      - unlabeled

concurrency:
  # One group per branch, so a run for staging does not cancel a run for main
  group: releaser-pleaser-${{ github.ref_name }}
  cancel-in-progress: true

jobs:
  releaser-pleaser:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    steps:
      - name: releaser-pleaser
        uses: apricote/releaser-pleaser@<version>
        with:
          branch: ${{ github.ref_name }}
          prerelease-id: ${{ github.ref_name == 'staging' && 'staging' || '' }}
```

For `pull_request_target` events, `github.ref_name` is the base branch of the pull request.

## Files in the release pull request

If you merge `main` into `staging` regularly, the release pull request for `staging` must not change files. Otherwise `CHANGELOG.md` and the version references conflict on each merge. For this reason, runs with `prerelease-id` do not run any [updaters](../reference/updaters.md) by default. The release pull request contains one commit without file changes, and the release notes are only in the pull request description and the release on the forge.

To change files on the pre-release branch anyway, enable the updaters explicitly, for example `updaters: changelog,generic`.

Forgejo and Gitea cannot squash-merge a pull request without file changes. On these forges, merge the release pull request of the pre-release branch with a merge commit instead.

## Versioning

- The version is calculated from the commits since the **last stable release**, the same as for the [pre-release labels](pre-releases.md#versioning).
- The counter at the end increases for each release on the pre-release branch: `v1.3.0-staging.0`, then `v1.3.0-staging.1`. When the calculated version changes, for example to `v1.4.0`, the counter starts again at `0`.
- The release notes contain the changes since the latest stable release or pre-release of the branch.
- When there are no changes since the latest pre-release of the branch, no release pull request is opened, and an open one is closed. Merge commits and the release commits of `releaser-pleaser` (for example the release commit of `main` after you merge `main` into `staging`) do not count as changes.
- The [`version-prefix`](../reference/github-action.md#inputs) applies to pre-releases too: with `version-prefix: none` the versions are `1.3.0-staging.0`, `1.3.0-staging.1`, ….
- Runs without `prerelease-id` ignore all tags with a custom identifier. Releases of `main` are therefore not affected by releases of `staging`: `main` still proposes `v1.3.0` after `v1.3.0-staging.2` was released, and its release notes contain all changes since the last stable release.
- Tags created through the `rp-next-version::alpha`, `rp-next-version::beta` and `rp-next-version::rc` labels are not affected by this.

On GitHub and Forgejo, the releases are marked as pre-releases. GitLab has no pre-release flag for releases, so the newest release of either branch is shown as the latest release.

The identifier must contain at least one letter and may only contain `0-9`, `A-Z`, `a-z` and `-`. The identifiers `alpha`, `beta` and `rc` are reserved for the [pre-release labels](pre-releases.md#creating-a-pre-release).

## Related Documentation

- **Guides**
  - [Pre-releases](pre-releases.md)
- **Reference**
  - [GitHub Action](../reference/github-action.md)
  - [GitLab CI/CD Component](../reference/gitlab-cicd-component.md)
