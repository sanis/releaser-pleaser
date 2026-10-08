package versioning

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/blang/semver/v4"

	"github.com/apricote/releaser-pleaser/internal/commitparser"
	"github.com/apricote/releaser-pleaser/internal/git"
)

var SemVer Strategy = semVer{}

// SemVerWithPrereleaseID returns a [Strategy] that always creates pre-releases with the given identifier,
// e.g. v1.2.0-staging.0. The next version type from the release pull request is ignored.
func SemVerWithPrereleaseID(id string) Strategy {
	return semVer{prereleaseID: id}
}

type semVer struct {
	prereleaseID string
}

func (s semVer) NextVersion(r git.Releases, versionBump VersionBump, nextVersionType NextVersionType) (string, error) {
	latest, err := parseSemverWithDefault(r.Latest)
	if err != nil {
		return "", fmt.Errorf("failed to parse latest version: %w", err)
	}

	stable, err := parseSemverWithDefault(r.Stable)
	if err != nil {
		return "", fmt.Errorf("failed to parse stable version: %w", err)
	}

	// If there is a previous stable release, we use that as the version anchor. Falling back to any pre-releases
	// if they are the only tags in the repo.
	next := latest
	if r.Stable != nil {
		next = stable
	}

	switch versionBump {
	case UnknownVersion:
		return "", fmt.Errorf("invalid latest bump (unknown)")
	case PatchVersion:
		err = next.IncrementPatch()
	case MinorVersion:
		err = next.IncrementMinor()
	case MajorVersion:
		err = next.IncrementMajor()
	}
	if err != nil {
		return "", err
	}

	if s.prereleaseID != "" {
		// Count up only from pre-releases of the same version, so the counter starts at 0 for each new version.
		if latest.Major != next.Major || latest.Minor != next.Minor || latest.Patch != next.Patch {
			latest.Pre = nil
		}

		err = setNextPRVersion(&next, latest, s.prereleaseID)
		if err != nil {
			return "", err
		}

		return "v" + next.String(), nil
	}

	switch nextVersionType {
	case NextVersionTypeUndefined, NextVersionTypeNormal:
		next.Pre = make([]semver.PRVersion, 0)
	case NextVersionTypeAlpha, NextVersionTypeBeta, NextVersionTypeRC:
		err = setNextPRVersion(&next, latest, nextVersionType.String())
		if err != nil {
			return "", err
		}
	}

	return "v" + next.String(), nil
}

// setNextPRVersion sets the pre-release of version to prType, counting up from the latest release if it is a
// pre-release of the same type.
func setNextPRVersion(version *semver.Version, latest semver.Version, prType string) error {
	id := uint64(0)

	if len(latest.Pre) >= 2 && latest.Pre[0].String() == prType {
		if latest.Pre[1].String() == "" || !latest.Pre[1].IsNumeric() {
			return fmt.Errorf("invalid format of previous tag")
		}
		id = latest.Pre[1].VersionNum + 1
	}

	setPRVersion(version, prType, id)
	return nil
}

func BumpFromCommits(commits []commitparser.AnalyzedCommit) VersionBump {
	bump := UnknownVersion

	for _, commit := range commits {
		entryBump := UnknownVersion
		switch {
		case commit.BreakingChange:
			entryBump = MajorVersion
		case commit.Type == "feat":
			entryBump = MinorVersion
		case commit.Type == "fix":
			entryBump = PatchVersion
		}

		if entryBump > bump {
			bump = entryBump
		}
	}

	return bump
}

func setPRVersion(version *semver.Version, prType string, count uint64) {
	version.Pre = []semver.PRVersion{
		{VersionStr: prType},
		{VersionNum: count, IsNum: true},
	}
}

func parseSemverWithDefault(tag *git.Tag) (semver.Version, error) {
	version := "v0.0.0"
	if tag != nil {
		version = tag.Name
	}

	// The lib can not handle v prefixes
	version = strings.TrimPrefix(version, "v")

	parsedVersion, err := semver.Parse(version)
	if err != nil {
		return semver.Version{}, fmt.Errorf("failed to parse version %q: %w", version, err)
	}

	return parsedVersion, nil
}

var prereleaseIDPattern = regexp.MustCompile(`^[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*$`)

// labelPrereleaseIDs are the pre-release identifiers that can be selected through the release pull request labels.
var labelPrereleaseIDs = []string{
	NextVersionTypeAlpha.String(),
	NextVersionTypeBeta.String(),
	NextVersionTypeRC.String(),
}

// ValidatePrereleaseID returns an error if id can not be used as the identifier for [SemVerWithPrereleaseID].
// An empty id is valid and disables the custom pre-release identifier.
func ValidatePrereleaseID(id string) error {
	if id == "" {
		return nil
	}

	if !prereleaseIDPattern.MatchString(id) {
		return fmt.Errorf("invalid pre-release identifier %q: must only contain alphanumerics and hyphens, and must not be numeric", id)
	}

	if slices.Contains(labelPrereleaseIDs, id) {
		return fmt.Errorf("invalid pre-release identifier %q: reserved for the release pull request labels", id)
	}

	return nil
}

// IncludesTag reports whether a release with the given version belongs to the releases that are considered when
// calculating the next version for the pre-release identifier prereleaseID.
//
// Stable releases are always included. Without a prereleaseID only pre-releases created through the release pull
// request labels (alpha, beta, rc) are included, otherwise only pre-releases with the same identifier.
func IncludesTag(version semver.Version, prereleaseID string) bool {
	if len(version.Pre) == 0 {
		return true
	}

	id := version.Pre[0].String()
	if prereleaseID == "" {
		return slices.Contains(labelPrereleaseIDs, id)
	}

	return id == prereleaseID
}

func (s semVer) IsPrerelease(version string) bool {
	semVersion, err := parseSemverWithDefault(&git.Tag{Hash: "", Name: version})
	if err != nil {
		return false
	}

	if len(semVersion.Pre) > 0 {
		return true
	}

	return false
}
