package changeset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChanduBobbili/changesetgoo/constants"
	"github.com/ChanduBobbili/changesetgoo/enums"
)

// ApplyChangesets merges all temp markdowns into CHANGELOG.md and clears .changesets
func ApplyChangesets() (string, error) {
	files, err := os.ReadDir(constants.ChangesDir)
	if err != nil {
		return "", fmt.Errorf("no changesets found")
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no changesets found")
	}

	// Track bump types and their messages separately
	var majors, minors, patches []string
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		data, _ := os.ReadFile(filepath.Join(constants.ChangesDir, file.Name()))
		lines := strings.Split(string(data), "\n")

		// ✅ Take only the first non-empty line as description
		var desc string
		for _, line := range lines {
			clean := strings.TrimSpace(strings.TrimPrefix(line, "###"))
			if clean != "" {
				desc = clean
				break
			}
		}
		if desc == "" {
			continue
		}

		name := strings.ToLower(file.Name())
		switch {
		case strings.HasPrefix(name, "major"):
			majors = append(majors, "- "+desc)
		case strings.HasPrefix(name, "minor"):
			minors = append(minors, "- "+desc)
		case strings.HasPrefix(name, "patch"):
			patches = append(patches, "- "+desc)
		}
	}

	// Decide bump type based on precedence
	var bumpType enums.ReleaseType
	switch {
	case len(majors) > 0:
		bumpType = enums.Major
	case len(minors) > 0:
		bumpType = enums.Minor
	case len(patches) > 0:
		bumpType = enums.Patch
	default:
		return "", fmt.Errorf("no valid bump types found in changesets")
	}

	// Get current version & bump
	current, _ := GetLatestVersion()
	newVersion, _ := BumpVersion(current, bumpType)

	// Build changelog entry
	var changelog strings.Builder
	changelog.WriteString(fmt.Sprintf("## %s\n\n", newVersion))

	if len(majors) > 0 {
		changelog.WriteString("### Major Changes\n\n")
		changelog.WriteString(strings.Join(majors, "\n"))
		changelog.WriteString("\n\n")
	}
	if len(minors) > 0 {
		changelog.WriteString("### Minor Changes\n\n")
		changelog.WriteString(strings.Join(minors, "\n"))
		changelog.WriteString("\n\n")
	}
	if len(patches) > 0 {
		changelog.WriteString("### Patch Changes\n\n")
		changelog.WriteString(strings.Join(patches, "\n"))
		changelog.WriteString("\n\n")
	}

	// Prepend changelog
	oldLog, _ := os.ReadFile("CHANGELOG.md")
	newLog := changelog.String() + string(oldLog)

	if err := os.WriteFile("CHANGELOG.md", []byte(newLog), 0644); err != nil {
		return "", err
	}

	// Cleanup changesets
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			os.Remove(filepath.Join(constants.ChangesDir, file.Name()))
		}
	}

	return newVersion, nil
}
