package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func isVersionCompatible(required string, current string) (bool, error) {
	required = strings.TrimLeftFunc(required, func(r rune) bool {
		return r != '=' && r != '^' && r != '~'
	})
	required = strings.TrimSpace(required)

	if len(required) == 0 {
		return false, errors.New("requirement prefix not found, expected '=', '^' or '~'")
	}

	current = strings.TrimLeftFunc(current, func(r rune) bool {
		return r != 'v'
	})
	current = strings.TrimSpace(current)
	if len(current) == 0 {
		return false, errors.New("version prefix not found, expected 'v'")
	}

	requirement := required[0]
	requiredVersion, err := parseSemanticVersion(required[1:])
	if err != nil {
		return false, fmt.Errorf("failed to parse required version: %w", err)
	}

	currentVersion, err := parseSemanticVersion(current[1:])
	if err != nil {
		return false, fmt.Errorf("failed to parse current version: %w", err)
	}

	switch requirement {
	case '=':
		return requiredVersion == currentVersion, nil
	case '^':
		if requiredVersion.major != currentVersion.major {
			return false, nil
		}
		if requiredVersion.minor > currentVersion.minor {
			return false, nil
		}
		if requiredVersion.minor == currentVersion.minor {
			return requiredVersion.patch <= currentVersion.patch, nil
		}
		return true, nil
	case '~':
		if requiredVersion.major != currentVersion.major || requiredVersion.minor != currentVersion.minor {
			return false, nil
		}
		return requiredVersion.patch <= currentVersion.patch, nil
	default:
		return false, fmt.Errorf("unknown required prefix %q", requirement)
	}
}

type SemVer struct {
	major int64
	minor int64
	patch int64
}

func parseSemanticVersion(version string) (SemVer, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return SemVer{}, errors.New("version must be in major.minor.patch format")
	}

	if major, err := strconv.ParseInt(parts[0], 10, 8); err != nil {
		return SemVer{}, errors.New("failed to parse major version")
	} else if minor, err := strconv.ParseInt(parts[1], 10, 8); err != nil {
		return SemVer{}, errors.New("failed to parse minor version")
	} else if patch, err := strconv.ParseInt(parts[2], 10, 8); err != nil {
		return SemVer{}, errors.New("failed to parse patch level")
	} else {
		return SemVer{major: major, minor: minor, patch: patch}, nil
	}
}
