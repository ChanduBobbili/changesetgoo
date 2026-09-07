package git

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type GitRepository struct {
	repo *git.Repository
}

func OpenGitRepo(repoPath *string) (*GitRepository, error) {
	repo, err := git.PlainOpen(*repoPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to open the git repository: %v", err)
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

	_, err = wt.Commit(message, &git.CommitOptions{})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

func (g *GitRepository) CheckTagExists(tagName string) (bool, error) {
	_, err := g.repo.Tag(tagName)
	if err == git.ErrTagNotFound {
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
	err := g.repo.Push(&git.PushOptions{
		Progress:   nil,
		FollowTags: true,
	})
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			return nil // No new commits or tags to push
		}
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
