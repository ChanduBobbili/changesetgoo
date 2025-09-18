package changeset

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChanduBobbili/changesetgoo/constants"
	"github.com/ChanduBobbili/changesetgoo/enums"
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
		return "", err
	}

	return enums.ReleaseType(result), nil
}

// AddChangeset creates a temp markdown file for the change
func AddChangeset(releaseType enums.ReleaseType, message string) error {
	if err := os.MkdirAll(constants.ChangesDir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s-%d.md", releaseType, time.Now().UnixNano())
	filepath := filepath.Join(constants.ChangesDir, filename)

	// ✅ Only write the plain description (no headings, no "###")
	content := strings.TrimSpace(message) + "\n"
	return os.WriteFile(filepath, []byte(content), 0644)
}

// InteractiveAdd allows user to input bump type and description
func InteractiveAdd() error {
	bump, err := PromptReleaseType()
	if err != nil {
		return err
	}

	fmt.Print("Enter change description: ")
	reader := bufio.NewReader(os.Stdin)
	desc, _ := reader.ReadString('\n')
	desc = strings.TrimSpace(desc)

	if err := AddChangeset(bump, desc); err != nil {
		return err
	}

	fmt.Println("✅ Changeset added")
	return nil
}
