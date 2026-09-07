package git

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ChanduBobbili/changesetgoo/utils"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type GitRepository struct {
	repo *git.Repository
}

func OpenGitRepo(repoPath *string) (*GitRepository, error) {
	repo, err := git.PlainOpenWithOptions(*repoPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, fmt.Errorf("failed to open the git repository: %v", err)
	}
	return &GitRepository{repo: repo}, nil
}

func (g *GitRepository) GetRepo() *git.Repository {
	return g.repo
}

func (g *GitRepository) GetGitUser() (userName string, userEmail string, err error) {
	config, err := g.repo.Config()
	if err != nil {
		return "", "", fmt.Errorf("failed to get git config: %w", err)
	}
	return config.User.Name, config.User.Email, nil
}

func (g *GitRepository) GetCurrentBranch() (string, error) {
	head, err := g.repo.Head()
	if err != nil {
		return "", fmt.Errorf("Error getting HEAD: %v", err)
	}

	branchName := head.Name().Short()
	return branchName, nil
}

func (g *GitRepository) GetRepoHead() (*plumbing.Reference, error) {
	return g.repo.Head()
}

func (g *GitRepository) GetRootCommit() (string, error) {
	isShallow, err := utils.ExecuteCommandOutput("git", "rev-parse", "--is-shallow-repository")
	if err == nil && strings.TrimSpace(isShallow) == "true" {
		return "", fmt.Errorf("repository is shallow; fetch full history before resolving root commit")
	}

	out, err := utils.ExecuteCommandOutput("git", "rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to resolve root commit: %w", err)
	}

	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return "", fmt.Errorf("no commits found in repository")
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", fmt.Errorf("no commits found in repository")
	}
	return parts[0], nil
}

func (g *GitRepository) DiffTreesFromRefs(fromRef string, toRef string) ([]string, error) {
	fromHash, err := g.repo.ResolveRevision(plumbing.Revision(fromRef))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", fromRef, err)
	}
	toHash, err := g.repo.ResolveRevision(plumbing.Revision(toRef))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", toRef, err)
	}

	fromCommit, err := g.repo.CommitObject(*fromHash)
	if err != nil {
		return nil, fmt.Errorf("failed to load source commit: %w", err)
	}
	toCommit, err := g.repo.CommitObject(*toHash)
	if err != nil {
		return nil, fmt.Errorf("failed to load target commit: %w", err)
	}

	fromTree, err := fromCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to load source tree: %w", err)
	}
	toTree, err := toCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to load target tree: %w", err)
	}

	changes, err := fromTree.Diff(toTree)
	if err != nil {
		return nil, fmt.Errorf("failed to diff trees: %w", err)
	}

	files := make([]string, 0, len(changes))
	seen := make(map[string]struct{}, len(changes))
	for _, change := range changes {
		name := ""
		if change.To.Name != "" {
			name = change.To.Name
		} else if change.From.Name != "" {
			name = change.From.Name
		}
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		files = append(files, name)
	}

	return files, nil
}

// AddFiles adds the specified files to the staging area (index) of the git repository.
// filePaths is a pointer to a slice of strings representing the file paths to be added.
// If filePaths is nil or empty, it will add all changes in the working directory.
func (g *GitRepository) AddFiles(filePaths *[]string) error {
	wt, err := g.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if filePaths == nil || len(*filePaths) == 0 {
		// Add all changes in the working directory
		return wt.AddWithOptions(&git.AddOptions{All: true})
	}

	// Add specified files
	for _, path := range *filePaths {
		if _, err := wt.Add(path); err != nil {
			return fmt.Errorf("failed to add file %s: %w", path, err)
		}
	}
	return nil
}

func (g *GitRepository) CommitChanges(message string) error {
	wt, err := g.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	userName, userEmail, err := g.GetGitUser()
	if err != nil {
		return fmt.Errorf("failed to get git user: %w", err)
	}
	if userName == "" {
		userName = "changesetgoo"
	}
	if userEmail == "" {
		userEmail = "changesetgoo@cli"
	}

	_, err = wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  userName,
			Email: userEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

func (g *GitRepository) CheckTagExists(tagName string) (bool, error) {
	_, err := g.repo.Tag(tagName)
	if errors.Is(err, git.ErrTagNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check tag existence: %w", err)
	}
	return true, nil
}

func (g *GitRepository) CreateTag(tagName string, message *string) error {
	userName, userEmail, err := g.GetGitUser()
	if err != nil {
		return fmt.Errorf("failed to get git user: %w", err)
	}
	// Fallback values in case the git config isn't globally or locally set
	if userName == "" {
		userName = "changesetgoo"
	}
	if userEmail == "" {
		userEmail = "changesetgoo@cli"
	}

	headRef, err := g.repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	tagOptions := &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  userName,
			Email: userEmail,
			When:  time.Now(),
		}}

	if message != nil {
		tagOptions.Message = *message
	}

	_, err = g.repo.CreateTag(tagName, headRef.Hash(), tagOptions)
	if err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	return nil
}

func (g *GitRepository) PushCommitsAndTags() error {
	if err := utils.ExecuteCommand("git", "push", "--follow-tags"); err != nil {
		return fmt.Errorf("failed to push commits and tags: %w", err)
	}
	return nil
}

func (g *GitRepository) HasChangesSinceRef(ref string) (bool, error) {
	// Resolve the target ref (e.g., "main", "HEAD~1", or a commit hash)
	targetHash, err := g.repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return false, fmt.Errorf("failed to resolve ref %s: %w", ref, err)
	}

	targetCommit, err := g.repo.CommitObject(*targetHash)
	if err != nil {
		return false, fmt.Errorf("failed to get target commit: %w", err)
	}
	targetTree, err := targetCommit.Tree()
	if err != nil {
		return false, err
	}

	// Resolve the current HEAD
	headRef, err := g.repo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get HEAD: %w", err)
	}
	headCommit, err := g.repo.CommitObject(headRef.Hash())
	if err != nil {
		return false, err
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return false, err
	}

	changes, err := targetTree.Diff(headTree)
	if err != nil {
		return false, fmt.Errorf("failed to diff trees: %w", err)
	}
	if len(changes) > 0 {
		return true, nil // There are committed differences
	}

	// Check the working directory for uncommitted modifications or untracked files
	// This replaces the 'git ls-files --others' and the uncommitted portion of 'git diff'
	wt, err := g.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree status: %w", err)
	}

	// IsClean() returns true only if the working tree exactly matches HEAD (no untracked, no modified)
	return !status.IsClean(), nil
}
