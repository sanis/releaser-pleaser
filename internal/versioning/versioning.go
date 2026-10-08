package versioning

import (
	"fmt"
	"strings"

	"github.com/leodido/go-conventionalcommits"

	"github.com/apricote/releaser-pleaser/internal/git"
)

type Strategy interface {
	NextVersion(git.Releases, VersionBump, NextVersionType) (string, error)
	IsPrerelease(version string) bool
}

// VPrefix is the "v" that most, but not all, repositories put in front of their version.
const VPrefix = "v"

// VersionPrefix decides if the emitted versions carry the VPrefix.
type VersionPrefix int

const (
	// VersionPrefixAuto reads the prefix from the tags that already exist in the repository. If
	// the repository has no tags yet, VPrefix is used.
	VersionPrefixAuto VersionPrefix = iota
	// VersionPrefixV always emits the VPrefix.
	VersionPrefixV
	// VersionPrefixNone never emits the VPrefix.
	VersionPrefixNone
)

func (p VersionPrefix) String() string {
	switch p {
	case VersionPrefixAuto:
		return "auto"
	case VersionPrefixV:
		return "v"
	case VersionPrefixNone:
		return "none"
	default:
		return ""
	}
}

// ParseVersionPrefix reads the --version-prefix flag. An empty value is treated like "auto", so
// that the forge templates can pass the flag through without setting a value.
func ParseVersionPrefix(input string) (VersionPrefix, error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "", "auto":
		return VersionPrefixAuto, nil
	case "v":
		return VersionPrefixV, nil
	case "none":
		return VersionPrefixNone, nil
	default:
		return VersionPrefixAuto, fmt.Errorf("unknown version prefix %q, expected one of: auto, v, none", input)
	}
}

type VersionBump conventionalcommits.VersionBump

const (
	UnknownVersion VersionBump = iota
	PatchVersion
	MinorVersion
	MajorVersion
)

type NextVersionType int

const (
	NextVersionTypeUndefined NextVersionType = iota
	NextVersionTypeNormal
	NextVersionTypeRC
	NextVersionTypeBeta
	NextVersionTypeAlpha
)

func (n NextVersionType) String() string {
	switch n {
	case NextVersionTypeUndefined:
		return "undefined"
	case NextVersionTypeNormal:
		return "normal"
	case NextVersionTypeRC:
		return "rc"
	case NextVersionTypeBeta:
		return "beta"
	case NextVersionTypeAlpha:
		return "alpha"
	default:
		return ""
	}
}

func (n NextVersionType) IsPrerelease() bool {
	switch n {
	case NextVersionTypeRC, NextVersionTypeBeta, NextVersionTypeAlpha:
		return true
	case NextVersionTypeUndefined, NextVersionTypeNormal:
		return false
	default:
		return false
	}
}
