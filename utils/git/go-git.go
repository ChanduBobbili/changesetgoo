package git

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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

func (g *GitRepository) GetCurrentBranch() (string, error) {
	head, err := g.repo.Head()
	if err != nil {
		return "", fmt.Errorf("Error getting HEAD: %v", err)
	}

	branchName := head.Name().Short()
	return branchName, nil
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
