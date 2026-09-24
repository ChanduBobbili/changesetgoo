package changeset

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ChanduBobbili/changesetgoo/config"
	"github.com/ChanduBobbili/changesetgoo/utils/git"
)

// GetChangedFiles returns the files changed between a ref and HEAD.
func GetChangedFiles(gitRepo *git.GitRepository, ref string) ([]string, error) {
	return gitRepo.DiffTreesFromRefs(ref, "HEAD")
}

func ResolveChangeRef(gitRepo *git.GitRepository, cfg config.Config) (string, error) {
	branch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		return "", err
	}

	if branch != cfg.BaseBranch {
		return cfg.BaseBranch, nil
	}

	version, err := GetLatestVersion()
	if err != nil {
		return "", err
	}
	tag := cfg.TagPrefix + version

	exists, err := gitRepo.CheckTagExists(tag)
	if err != nil {
		return "", fmt.Errorf("failed to check tag %s: %w", tag, err)
	}
	if exists {
		return tag, nil
	}

	rootCommit, err := gitRepo.GetRootCommit()
	if err != nil {
		return "", fmt.Errorf("no tag %s found and failed to resolve root commit: %w", tag, err)
	}
	return rootCommit, nil
}

func HasChangesForChangeset(gitRepo *git.GitRepository, cfg config.Config) (bool, error) {
	ref, err := ResolveChangeRef(gitRepo, cfg)
	if err != nil {
		return false, err
	}
	return gitRepo.HasChangesSinceRef(ref)
}

// GetRelevantChangedFiles returns changed files matching ChangedFilePatterns.
func GetRelevantChangedFiles(gitRepo *git.GitRepository, cfg config.Config) ([]string, error) {
	ref, err := ResolveChangeRef(gitRepo, cfg)
	if err != nil {
		return nil, err
	}

	files, err := GetChangedFiles(gitRepo, ref)
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
func CheckChangesetRequirement(gitRepo *git.GitRepository, cfg config.Config) (bool, []string, error) {
	relevantFiles, err := GetRelevantChangedFiles(gitRepo, cfg)
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
