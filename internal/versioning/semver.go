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

// SemVer returns the semver strategy. prefix decides if the emitted versions carry the VPrefix.
func SemVer(prefix VersionPrefix) Strategy {
	return semVer{prefix: prefix}
}

// SemVerWithPrereleaseID returns a [Strategy] that always creates pre-releases with the given identifier,
// e.g. v1.2.0-staging.0. The next version type from the release pull request is ignored. prefix decides if the
// emitted versions carry the VPrefix.
func SemVerWithPrereleaseID(id string, prefix VersionPrefix) Strategy {
	return semVer{prefix: prefix, prereleaseID: id}
}

type semVer struct {
	prefix       VersionPrefix
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
	// if they are the only tags in the repo. With a prereleaseID, the latest release is always one of our own
	// pre-releases, so we anchor on the stable release (or v0.0.0) and count up the pre-release instead.
	next := latest
	if r.Stable != nil || s.prereleaseID != "" {
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

		return s.versionPrefix(r) + next.String(), nil
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

	return s.versionPrefix(r) + next.String(), nil
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

// versionPrefix returns the prefix to put in front of the version. With VersionPrefixAuto we keep
// whatever convention the repository already uses, so that a repository tagged "1.2.3" does not
// switch to "v1.2.4" halfway through its history.
func (s semVer) versionPrefix(r git.Releases) string {
	switch s.prefix {
	case VersionPrefixV:
		return VPrefix
	case VersionPrefixNone:
		return ""
	case VersionPrefixAuto:
		// The stable release is the better anchor, we only look at prereleases if the repository
		// has never had a stable release.
		for _, tag := range []*git.Tag{r.Stable, r.Latest} {
			if tag == nil {
				continue
			}

			if strings.HasPrefix(tag.Name, VPrefix) {
				return VPrefix
			}

			return ""
		}
	}

	// No tag to read the convention from. Keep the "v" that releaser-pleaser has always emitted.
	return VPrefix
}

// BumpFromCommits returns the version bump required for the commits. Every commit is releasable,
// types without a dedicated meaning (chore, docs, other, ...) result in a patch release.
func BumpFromCommits(commits []commitparser.AnalyzedCommit) VersionBump {
	bump := UnknownVersion

	for _, commit := range commits {
		entryBump := PatchVersion
		switch {
		case commit.BreakingChange:
			entryBump = MajorVersion
		case commit.Type == commitparser.TypeFeature:
			entryBump = MinorVersion
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
