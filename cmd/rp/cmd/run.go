package cmd

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	rp "github.com/apricote/releaser-pleaser"
	"github.com/apricote/releaser-pleaser/internal/commitparser/conventionalcommits"
	"github.com/apricote/releaser-pleaser/internal/forge"
	"github.com/apricote/releaser-pleaser/internal/forge/forgejo"
	"github.com/apricote/releaser-pleaser/internal/forge/github"
	"github.com/apricote/releaser-pleaser/internal/forge/gitlab"
	"github.com/apricote/releaser-pleaser/internal/log"
	"github.com/apricote/releaser-pleaser/internal/updater"
	"github.com/apricote/releaser-pleaser/internal/versioning"
)

func newRunCommand() *cobra.Command {
	var (
		flagForge      string
		flagBranch     string
		flagOwner      string
		flagRepo       string
		flagExtraFiles string
		flagUpdaters   []string

		flagPrereleaseID string

		flagAPIURL   string
		flagAPIToken string
		flagUsername string
	)

	cmd := &cobra.Command{
		Use: "run",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			logger := log.GetLogger(cmd.ErrOrStderr())

			var err error

			logger.DebugContext(ctx, "run called",
				"forge", flagForge,
				"branch", flagBranch,
				"owner", flagOwner,
				"repo", flagRepo,
				"prerelease-id", flagPrereleaseID,
			)

			err = versioning.ValidatePrereleaseID(flagPrereleaseID)
			if err != nil {
				return err
			}

			versioningStrategy := versioning.SemVer
			if flagPrereleaseID != "" {
				versioningStrategy = versioning.SemVerWithPrereleaseID(flagPrereleaseID)
			}

			var f forge.Forge

			forgeOptions := forge.Options{
				Repository:   flagRepo,
				BaseBranch:   flagBranch,
				PrereleaseID: flagPrereleaseID,
			}

			switch flagForge {
			case "gitlab":
				logger.DebugContext(ctx, "using forge GitLab")
				f, err = gitlab.New(logger, &gitlab.Options{
					Options: forgeOptions,
					Path:    fmt.Sprintf("%s/%s", flagOwner, flagRepo),
				})
				if err != nil {
					slog.ErrorContext(ctx, "failed to create client", "err", err)
					return fmt.Errorf("failed to create gitlab client: %w", err)
				}
			case "github":
				logger.DebugContext(ctx, "using forge GitHub")
				f = github.New(logger, &github.Options{
					Options: forgeOptions,
					Owner:   flagOwner,
					Repo:    flagRepo,
				})
			case "forgejo":
				logger.DebugContext(ctx, "using forge Forgejo")
				f, err = forgejo.New(logger, &forgejo.Options{
					Options: forgeOptions,
					Owner:   flagOwner,
					Repo:    flagRepo,

					APIURL:   flagAPIURL,
					APIToken: flagAPIToken,
					Username: flagUsername,
				})
				if err != nil {
					logger.ErrorContext(ctx, "failed to create client", "err", err)
					return fmt.Errorf("failed to create forgejo client: %w", err)
				}
			default:
				return fmt.Errorf("unknown --forge: %s", flagForge)
			}

			extraFiles := parseExtraFiles(flagExtraFiles)

			updaterNames := parseUpdaters(flagUpdaters, flagPrereleaseID)
			updaters := []updater.Updater{}
			for _, name := range updaterNames {
				switch name {
				case "generic":
					updaters = append(updaters, updater.Generic(extraFiles))
				case "changelog":
					updaters = append(updaters, updater.Changelog())
				case "packagejson":
					updaters = append(updaters, updater.PackageJson())
				default:
					return fmt.Errorf("unknown updater: %s", name)
				}
			}

			releaserPleaser := rp.New(
				f,
				logger,
				flagBranch,
				conventionalcommits.NewParser(logger),
				versioningStrategy,
				extraFiles,
				updaters,
			)

			return releaserPleaser.Run(ctx)
		},
	}

	cmd.PersistentFlags().StringVar(&flagForge, "forge", "", "")
	cmd.PersistentFlags().StringVar(&flagBranch, "branch", "main", "")
	cmd.PersistentFlags().StringVar(&flagOwner, "owner", "", "")
	cmd.PersistentFlags().StringVar(&flagRepo, "repo", "", "")
	cmd.PersistentFlags().StringVar(&flagExtraFiles, "extra-files", "", "")
	cmd.PersistentFlags().StringSliceVar(&flagUpdaters, "updaters", []string{}, "")
	cmd.PersistentFlags().StringVar(&flagPrereleaseID, "prerelease-id", "", "Release every version on --branch as a pre-release with this identifier, e.g. staging -> v1.2.0-staging.0. Disables the default updaters.")

	cmd.PersistentFlags().StringVar(&flagAPIURL, "api-url", "", "")
	cmd.PersistentFlags().StringVar(&flagAPIToken, "api-token", "", "")
	cmd.PersistentFlags().StringVar(&flagUsername, "username", "", "")

	return cmd
}

func parseExtraFiles(input string) []string {
	// We quote the arg to avoid issues with the expected newlines in the value.
	// Need to remove those quotes before parsing the data
	input = strings.Trim(input, `"`)
	// In some situations we get a "\n" sequence, where we actually expect new lines,
	// replace the two characters with an actual new line
	input = strings.ReplaceAll(input, `\n`, "\n")
	lines := strings.Split(input, "\n")

	extraFiles := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			extraFiles = append(extraFiles, line)
		}
	}

	return extraFiles
}

// parseUpdaters returns the names of the updaters to run. Pre-release branches (prereleaseID set) do not run any
// updaters by default, so their release pull requests do not change files that conflict with merges from the
// stable branch.
func parseUpdaters(input []string, prereleaseID string) []string {
	names := []string{"changelog", "generic"}
	if prereleaseID != "" {
		names = []string{}
	}

	for _, u := range input {
		if u == "" {
			continue
		}

		if strings.HasPrefix(u, "-") {
			name := u[1:]
			names = slices.DeleteFunc(names, func(existingName string) bool { return existingName == name })
		} else {
			names = append(names, u)
		}
	}

	// Make sure we only have unique updaters
	slices.Sort(names)
	names = slices.Compact(names)

	return names
}
