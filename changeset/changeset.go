package changeset

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChanduBobbili/changesetgoo/config"
	"github.com/ChanduBobbili/changesetgoo/enums"
	"github.com/ChanduBobbili/changesetgoo/utils/git"
	"github.com/chzyer/readline"
	"github.com/manifoldco/promptui"
)

// PromptReleaseType asks user to select release type
func PromptReleaseType() (enums.ReleaseType, error) {
	prompt := promptui.Select{
		Label: "Select release type",
		Items: []string{"major", "minor", "patch"},
	}

	_, result, err := prompt.Run()
	if err != nil {
		// Handle graceful shutdown for Ctrl+C and Ctrl+D
		if errors.Is(err, promptui.ErrInterrupt) || errors.Is(err, promptui.ErrEOF) {
			os.Exit(0)
		}
		return "", err
	}

	return enums.ReleaseType(result), nil
}

// AddChangeset creates a temp markdown file for the change.
func AddChangeset(releaseType enums.ReleaseType, message string, cfg config.Config) error {
	if err := os.MkdirAll(cfg.ChangesDir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s-%d.md", releaseType, time.Now().UnixNano())
	filePath := filepath.Join(cfg.ChangesDir, filename)

	// Only write the plain description (no headings, no "###").
	content := strings.TrimSpace(message) + "\n"
	return os.WriteFile(filePath, []byte(content), 0644)
}

// InteractiveAdd allows user to input bump type and description
func InteractiveAdd(gitRepo *git.GitRepository, cfg config.Config) error {
	hasChanges, err := HasChangesForChangeset(gitRepo, cfg)
	if err != nil {
		return err
	}
	if !hasChanges {
		return fmt.Errorf("no changes detected; nothing to add a changeset for")
	}

	bump, err := PromptReleaseType()
	if err != nil {
		return err
	}

	// Use readline directly to fix the multi-print pasting bug
	rl, err := readline.New("Enter change description: ")
	if err != nil {
		return err
	}
	defer rl.Close()

	desc, err := rl.Readline()
	if err != nil {
		// Handle graceful shutdown for Ctrl+C and Ctrl+D during text input
		if errors.Is(err, readline.ErrInterrupt) || errors.Is(err, io.EOF) {
			os.Exit(0)
		}
		return err
	}

	desc = strings.TrimSpace(desc)

	if err := AddChangeset(bump, desc, cfg); err != nil {
		return err
	}

	return nil
}
