package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ChanduBobbili/changesetgoo/changeset"
	"github.com/ChanduBobbili/changesetgoo/config"
	"github.com/ChanduBobbili/changesetgoo/constants"
	"github.com/ChanduBobbili/changesetgoo/enums"
)

var (
	flagYes  bool
	flagPush bool
	flagCheck bool
)

func main() {
	// Default values for flags
	flagYes = false
	flagPush = false
	flagCheck = false

	args := os.Args[1:]
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	// First arg is the subcommand
	cmd := args[0]

	// Parse flags that appear after the subcommand
	for _, arg := range args[1:] {
		switch arg {
		case "--push":
			flagPush = true
		case "--yes":
			flagYes = true
		case "--check":
			flagCheck = true
		default:
			// If it's unknown, ignore or handle positional args
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Failed to load config: %v. Using defaults.\n", err)
		cfg = config.Defaults()
	}

	// Handle subcommands
	switch cmd {
	case "add":
		runAdd(cfg)
	case "version":
		runVersion(cfg)
	case "tag":
		runTag(cfg)
	case "status":
		runStatus(cfg)
	case "publish":
		runPublish(cfg)
	case "--version", "-v":
		printCLIVersion()
	case "help", "--help", "-h":
		printUsage()
		os.Exit(0)
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(2)
	}
}

func runAdd(cfg config.Config) {
	if err := changeset.InteractiveAdd(cfg); err != nil {
		fmt.Println("⚠️ Failed to add changeset:", err)
		os.Exit(1)
	}
	fmt.Println("✅ Changeset added")
	os.Exit(0)
}

func runVersion(cfg config.Config) {
	newVer, err := changeset.ApplyChangesets(cfg)
	if err != nil {
		fmt.Println("⚠️", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Version bumped to %s%s\n", cfg.TagPrefix, newVer)
	os.Exit(0)
}

func runTag(cfg config.Config) {
	version, err := changeset.GetLatestVersion()
	if err != nil {
		fmt.Println("⚠️ Failed to get latest version:", err)
		os.Exit(1)
	}

	tagName := cfg.TagPrefix + version
	checkCmd := exec.Command("git", "tag", "--list", tagName)
	out, _ := checkCmd.Output()
	if string(out) != "" {
		fmt.Printf("⚠️ Tag %s already exists, skipping.\n", tagName)
		os.Exit(0)
	}

	createTag(tagName, cfg.TagPrefix)

	if flagPush {
		pushTags()
	}
	os.Exit(0)
}

func runPublish(cfg config.Config) {
	if flagCheck {
		passes, relevantFiles, err := changeset.CheckChangesetRequirement(cfg)
		if err != nil {
			fmt.Println("⚠️ Failed to validate changeset requirement:", err)
			os.Exit(1)
		}
		if !passes {
			fmt.Println("⚠️ Relevant changes detected but no pending changeset was found")
			fmt.Println("Run: changesetgoo add")
			for _, file := range relevantFiles {
				fmt.Println("  -", file)
			}
			os.Exit(1)
		}
	}

	nextVer, bumpType, err := changeset.CalculateNextVersion(cfg)
	if err != nil {
		fmt.Println("⚠️", err)
		os.Exit(1)
	}

	previewRelease(nextVer, bumpType, cfg.TagPrefix)

	if !flagYes {
		confirmRelease()
	}

	tagName := bumpVersion(cfg)
	if cfg.Commit.Enabled {
		commitChanges(tagName, cfg)
	}

	createTag(tagName, cfg.TagPrefix)

	if flagPush {
		pushTags()
	}

	fmt.Printf("🎉 Published: %s\n", tagName)
	os.Exit(0)
}

func runStatus(cfg config.Config) {
	passes, relevantFiles, err := changeset.CheckChangesetRequirement(cfg)
	if err != nil {
		fmt.Println("⚠️ Failed to validate changeset requirement:", err)
		os.Exit(1)
	}

	if passes && len(relevantFiles) == 0 {
		fmt.Printf("✅ No changes matched changedFilePatterns against %s\n", cfg.BaseBranch)
		os.Exit(0)
	}
	if passes {
		fmt.Println("✅ Relevant changes detected and pending changesets are present")
		os.Exit(0)
	}

	fmt.Println("⚠️ Relevant changes detected but no pending changeset was found")
	fmt.Println("Run: changesetgoo add")
	for _, file := range relevantFiles {
		fmt.Println("  -", file)
	}
	os.Exit(1)
}

func previewRelease(nextVer string, bumpType enums.ReleaseType, tagPrefix string) {
	fmt.Println("📦 Release preview")
	fmt.Println("------------------")
	fmt.Printf(" Pending bump : %s\n", bumpType)
	fmt.Printf(" Next version : %s%s\n\n", tagPrefix, nextVer)
}

func confirmRelease() {
	fmt.Print("Do you want to continue with this release? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		fmt.Println("❌ Publish cancelled.")
		os.Exit(2)
	}
}

func bumpVersion(cfg config.Config) string {
	newVer, err := changeset.ApplyChangesets(cfg)
	if err != nil {
		fmt.Println("⚠️", err)
		os.Exit(1)
	}
	tagName := cfg.TagPrefix + newVer
	fmt.Printf("✅ Version bumped: %s\n", tagName)
	return tagName
}

func commitChanges(tagName string, cfg config.Config) {
	version := strings.TrimPrefix(tagName, cfg.TagPrefix)
	commitMessage := config.Render(cfg.Commit.Message, map[string]string{"tag": tagName, "version": version})

	if err := runCmd("git", "add", "-A"); err != nil {
		fmt.Println("⚠️ No changes to commit.")
	} else if err := runCmd("git", "commit", "-m", commitMessage); err != nil {
		fmt.Println("⚠️ No changes to commit.")
	} else {
		fmt.Printf("✅ Committed release changes: %s\n", commitMessage)
	}
}

func createTag(tagName string, tagPrefix string) {
	message := getChangelogForTag(tagName, tagPrefix)
	if err := runCmd("git", "tag", "-a", tagName, "-m", message); err != nil {
		fmt.Println("⚠️ Failed to create tag:", err)
		os.Exit(3)
	}
	fmt.Printf("✅ Git tag %s created\n", tagName)
}

func pushTags() {
	if err := runCmd("git", "push", "--follow-tags"); err != nil {
		fmt.Println("⚠️ Failed to push changes:", err)
		os.Exit(3)
	}
	fmt.Println("✅ Changes pushed with tags")
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getChangelogForTag(tagName string, tagPrefix string) string {
	// tagName is prefix+version, version is semver only.
	version := strings.TrimPrefix(tagName, tagPrefix)
	baseMessage := "Release " + tagName

	data, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		return baseMessage // Return base message if changelog can't be read
	}

	// (?s) allows . to match newlines. Stop at the next ## or end of file.
	re := regexp.MustCompile(fmt.Sprintf(`(?s)##\s+%s\s*\n(.*?)(?:\n##\s|\z)`, regexp.QuoteMeta(version)))
	matches := re.FindStringSubmatch(string(data))

	if len(matches) > 1 {
		return baseMessage + "\n\n" + strings.TrimSpace(matches[1])
	}

	return baseMessage // Return base message if no specific changelog is found
}

func printUsage() {
	fmt.Println("Usage: changesetgoo <command> [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  add         Add a new changeset interactively")
	fmt.Println("  version     Apply pending changesets and bump version")
	fmt.Println("  tag         Create a git tag for the latest version")
	fmt.Println("  status      Check changed files against changedFilePatterns")
	fmt.Println("  publish     Bump version, commit, and create a tag")
	fmt.Println("  help        Show this help message")
	fmt.Println("  --version, -v    Show changesetgoo CLI version")
	fmt.Println("\nFlags:")
	fmt.Println("  --yes            Auto-confirm publish without prompting")
	fmt.Println("  --push           Auto-push commits and tags after publish")
	fmt.Println("  --check          Enforce changedFilePatterns changeset requirement")
}

func printCLIVersion() {
	fmt.Println("changesetgoo", constants.CliVersion)
}
