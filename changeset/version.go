package changeset

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ChanduBobbili/changesetgoo/constants"
	"github.com/ChanduBobbili/changesetgoo/enums"
)

// GetLatestVersion reads CHANGELOG.md and returns the latest semantic version
func GetLatestVersion() (string, error) {
	data, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		return "0.0.0", nil
	}

	// Allow "## 1.0.0" or "## v1.0.0"
	re := regexp.MustCompile(`##\s+v?(\d+\.\d+\.\d+)`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	if len(matches) == 0 {
		return "0.0.0", nil
	}

	// Prepending changelogs means newest is always first
	return matches[0][1], nil
}

// BumpVersion returns the new version string given a bump type
func BumpVersion(current string, releaseType enums.ReleaseType) (string, error) {
	parts := strings.Split(current, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version format: %s", current)
	}

	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	patch, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return "", fmt.Errorf("invalid version number: %s", current)
	}

	switch releaseType {
	case enums.Major:
		major++
		minor, patch = 0, 0
	case enums.Minor:
		minor++
		patch = 0
	case enums.Patch:
		patch++
	default:
		return "", fmt.Errorf("unknown release type: %s", releaseType)
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

// CalculateNextVersion inspects pending changesets and returns the highest bump type + next version
func CalculateNextVersion() (string, enums.ReleaseType, error) {
	files, err := os.ReadDir(constants.ChangesDir)
	if err != nil || len(files) == 0 {
		return "", "", fmt.Errorf("no changesets found")
	}

	bumpType, err := detectBumpType(files)
	if err != nil {
		return "", "", err
	}

	current, _ := GetLatestVersion()
	nextVersion, _ := BumpVersion(current, bumpType)

	return nextVersion, bumpType, nil
}

func detectBumpType(files []os.DirEntry) (enums.ReleaseType, error) {
	hasMajor, hasMinor, hasPatch := false, false, false

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := strings.ToLower(file.Name())
		switch {
		case strings.HasPrefix(name, "major"):
			hasMajor = true
		case strings.HasPrefix(name, "minor"):
			hasMinor = true
		case strings.HasPrefix(name, "patch"):
			hasPatch = true
		}
	}

	switch {
	case hasMajor:
		return enums.Major, nil
	case hasMinor:
		return enums.Minor, nil
	case hasPatch:
		return enums.Patch, nil
	default:
		return "", fmt.Errorf("no valid bump types found in changesets")
	}
}
