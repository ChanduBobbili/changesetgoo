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
	if err != nil || len(files) == 0 {
		return "", fmt.Errorf("no changesets found")
	}

	majors, minors, patches := categorizeChangesets(files)
	if len(majors)+len(minors)+len(patches) == 0 {
		return "", fmt.Errorf("no valid bump types found in changesets")
	}

	bumpType := determineBumpType(majors, minors, patches)

	current, _ := GetLatestVersion()
	newVersion, _ := BumpVersion(current, bumpType)

	if err := updateChangelog(newVersion, majors, minors, patches); err != nil {
		return "", err
	}

	if err := cleanupChangesets(files); err != nil {
		return "", err
	}

	return newVersion, nil
}

func categorizeChangesets(files []os.DirEntry) ([]string, []string, []string) {
	var majors, minors, patches []string

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}
		desc := extractDescription(filepath.Join(constants.ChangesDir, file.Name()))
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

	return majors, minors, patches
}

func extractDescription(path string) string {
	data, _ := os.ReadFile(path)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		clean := strings.TrimSpace(strings.TrimPrefix(line, "###"))
		if clean != "" {
			return clean
		}
	}
	return ""
}

func determineBumpType(majors, minors, patches []string) enums.ReleaseType {
	switch {
	case len(majors) > 0:
		return enums.Major
	case len(minors) > 0:
		return enums.Minor
	default:
		return enums.Patch
	}
}

func updateChangelog(newVersion string, majors, minors, patches []string) error {
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

	oldLog, _ := os.ReadFile("CHANGELOG.md")
	newLog := changelog.String() + string(oldLog)
	return os.WriteFile("CHANGELOG.md", []byte(newLog), 0644)
}

func cleanupChangesets(files []os.DirEntry) error {
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			if err := os.Remove(filepath.Join(constants.ChangesDir, file.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
