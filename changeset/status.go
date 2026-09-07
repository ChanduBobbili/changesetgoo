package changeset

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ChanduBobbili/changesetgoo/config"
	"github.com/ChanduBobbili/changesetgoo/utils/git"
)

// GetChangedFiles returns the files changed relative to baseBranch.
func GetChangedFiles(baseBranch string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", baseBranch+"...HEAD")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to diff against %s: %w", baseBranch, err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return []string{}, nil
	}

	lines := strings.Split(trimmed, "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

func ResolveChangeRef(gitRepo *git.GitRepository, cfg config.Config) (string, error) {
	branch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		return "", err
	}

	// feature branch — compare against base branch
	if branch != cfg.BaseBranch {
		return cfg.BaseBranch, nil
	}

	// on base branch — compare against latest released tag.
	version, err := GetLatestVersion()
	if err != nil {
		return "", err
	}
	tag := cfg.TagPrefix + version

	verifyCmd := exec.Command("git", "rev-parse", "--verify", "--quiet", tag)
	if err := verifyCmd.Run(); err == nil {
		return tag, nil
	}

	// No tag yet — fall back to the repo's root commit.
	rootOut, err := exec.Command("git", "rev-list", "--max-parents=0", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("no tag %s found and failed to resolve root commit: %w", tag, err)
	}
	rootCommits := strings.Fields(strings.TrimSpace(string(rootOut)))
	if len(rootCommits) == 0 {
		return "", fmt.Errorf("no tag %s found and no root commit resolved", tag)
	}
	// Use the first root commit if there are multiple (unrelated histories).
	return rootCommits[0], nil
}

func HasChangesForChangeset(gitRepo *git.GitRepository, cfg config.Config) (bool, error) {
	ref, err := ResolveChangeRef(gitRepo, cfg)
	if err != nil {
		return false, err
	}
	return gitRepo.HasChangesSinceRef(ref)
}

// GetRelevantChangedFiles returns changed files matching ChangedFilePatterns.
func GetRelevantChangedFiles(cfg config.Config) ([]string, error) {
	files, err := GetChangedFiles(cfg.BaseBranch)
	if err != nil {
		return nil, err
	}

	relevant := make([]string, 0)
	for _, file := range files {
		if MatchesAnyPattern(file, cfg.ChangedFilePatterns) {
			relevant = append(relevant, file)
		}
	}

	return relevant, nil
}

// MatchesAnyPattern reports whether a file matches at least one pattern.
func MatchesAnyPattern(file string, patterns []string) bool {
	for _, pattern := range patterns {
		if globToRegexp(pattern).MatchString(file) {
			return true
		}
	}
	return false
}

func globToRegexp(pattern string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")

	for i := 0; i < len(pattern); {
		switch {
		case strings.HasPrefix(pattern[i:], "**"):
			b.WriteString(".*")
			i += 2
		case pattern[i] == '*':
			b.WriteString("[^/]*")
			i++
		case pattern[i] == '?':
			b.WriteString(".")
			i++
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
			i++
		}
	}

	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

// HasPendingChangesets reports whether there are any pending .md changesets.
func HasPendingChangesets(changesDir string) (bool, error) {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			return true, nil
		}
	}

	return false, nil
}

// CheckChangesetRequirement enforces changedFilePatterns semantics for CI-style
// checks. It returns whether the check passes and the relevant matched files.
func CheckChangesetRequirement(cfg config.Config) (bool, []string, error) {
	relevantFiles, err := GetRelevantChangedFiles(cfg)
	if err != nil {
		return false, nil, err
	}

	if len(relevantFiles) == 0 {
		return true, relevantFiles, nil
	}

	hasPending, err := HasPendingChangesets(cfg.ChangesDir)
	if err != nil {
		return false, nil, err
	}

	if hasPending {
		return true, relevantFiles, nil
	}

	return false, relevantFiles, nil
}
