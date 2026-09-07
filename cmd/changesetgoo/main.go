package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ChanduBobbili/changesetgoo/changeset"
	"github.com/ChanduBobbili/changesetgoo/config"
	"github.com/ChanduBobbili/changesetgoo/constants"
	"github.com/ChanduBobbili/changesetgoo/enums"
	"github.com/ChanduBobbili/changesetgoo/utils/exits"
	"github.com/ChanduBobbili/changesetgoo/utils/git"
)

func main() {
	// Define command-line flags
	repoPath := flag.String("repo", ".", "Path to the target git repository")
	flagYes := flag.Bool("yes", false, "Auto-confirm publish without prompting")
	flagPush := flag.Bool("push", false, "Push changes after publishing")
	flagCheck := flag.Bool("check", false, "Check changeset requirements")

	// Custom usage function to display help
	flag.Usage = func() {
		exits.WithInfo(getUsage())
	}
	// Parse command-line flags
	flag.Parse()

	args := os.Args[1:]
	if len(args) < 1 {
		exits.WithInfo(getUsage())
	}

	// Open the git repository
	gitRepo, err := git.OpenGitRepo(repoPath)
	if err != nil {
		exits.WithGitError("%v", err)
	}

	// First arg is the command
	cmd := args[0]

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Failed to load config: %v. Using defaults.\n", err)
		cfg = config.Defaults()
	}

	// Handle commands
	switch cmd {
	case "add":
		runAdd(cfg)
	case "version":
		runVersion(cfg)
	case "tag":
		runTag(cfg, *flagPush)
	case "status":
		runStatus(cfg)
	case "publish":
		runPublish(cfg, *flagYes, *flagPush, *flagCheck)
	case "--version", "-v":
		printCLIVersion()
	case "help", "--help", "-h":
		exits.WithInfo(getUsage())
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		exits.WithUsageError(getUsage())
	}
}

func runAdd(cfg config.Config) {
	if err := changeset.InteractiveAdd(cfg); err != nil {
		exits.WithError("⚠️ Failed to add changeset: %v", err)
	}
	exits.WithSuccess("✅ Changeset added")
}

func runVersion(cfg config.Config) {
	newVer, err := changeset.ApplyChangesets(cfg)
	if err != nil {
		exits.WithError("⚠️ %v", err)
	}
	exits.WithSuccess("✅ Version bumped to %s%s", cfg.TagPrefix, newVer)
}

func runTag(cfg config.Config, flagPush bool) {
	version, err := changeset.GetLatestVersion()
	if err != nil {
		exits.WithError("⚠️ Failed to get latest version: %v", err)
	}

	tagName := cfg.TagPrefix + version
	checkCmd := exec.Command("git", "tag", "--list", tagName)
	out, err := checkCmd.Output()
	if err != nil {
		exits.WithError("⚠️ Failed to check existing tags: %v", err)
	}
	if string(out) != "" {
		exits.WithInfo("⚠️ Tag %s already exists, skipping.", tagName)
	}

	createTag(tagName, cfg.TagPrefix)

	if flagPush {
		pushTags()
	}
	exits.WithSuccess("✅ Tag created: %s", tagName)
}

func runPublish(cfg config.Config, flagYes bool, flagPush bool, flagCheck bool) {
	if flagCheck {
		passes, relevantFiles, err := changeset.CheckChangesetRequirement(cfg)
		if err != nil {
			exits.WithError("⚠️ Failed to validate changeset requirement: %v", err)
		}
		if !passes {
			for _, file := range relevantFiles {
				fmt.Println("  -", file)
			}
			exits.WithInfo("⚠️ Relevant changes detected but no pending changeset was found\n Run: changesetgoo add")
		}
	}

	nextVer, bumpType, err := changeset.CalculateNextVersion(cfg)
	if err != nil {
		exits.WithError("⚠️ Failed to calculate next version: %v", err)
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

	exits.WithSuccess("🎉 Published: %s\n", tagName)
}

func runStatus(cfg config.Config) {
	passes, relevantFiles, err := changeset.CheckChangesetRequirement(cfg)
	if err != nil {
		exits.WithError("⚠️ Failed to validate changeset requirement: %v", err)
	}

	if passes && len(relevantFiles) == 0 {
		exits.WithSuccess("✅ No changes matched changedFilePatterns against %s\n", cfg.BaseBranch)
	}
	if passes {
		exits.WithSuccess("✅ Relevant changes detected and pending changesets are present")
	}

	for _, file := range relevantFiles {
		fmt.Println("  -", file)
	}
	exits.WithInfo("⚠️ Relevant changes detected but no pending changeset was found\n Run: changesetgoo add")
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

func getUsage() string {
	return `Usage: changesetgoo <command> [flags]

Commands:
  add        Add a new changeset interactively
  version    Apply pending changesets and bump version
  tag        Create a git tag for the latest version
  status     Check changed files against changedFilePatterns
  publish    Bump version, commit, and create a tag
  help       Show this help message
  --version, -v    Show changesetgoo CLI version

Flags:
  --yes            Auto-confirm publish without prompting
  --push           Auto-push commits and tags after publish
  --check          Enforce changedFilePatterns changeset requirement
`
}

func printCLIVersion() {
	fmt.Println("changesetgoo", constants.CliVersion)
}
