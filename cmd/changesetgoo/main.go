package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/ChanduBobbili/changesetgoo/changeset"
	"github.com/ChanduBobbili/changesetgoo/enums"
)

var (
	flagYes  bool
	flagPush bool
)

func main() {
	// Default values for flags
	flagYes = false
	flagPush = false

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
		default:
			// If it's unknown, ignore or handle positional args
		}
	}

	// Handle subcommands
	switch cmd {
	case "add":
		runAdd()
	case "version":
		runVersion()
	case "tag":
		runTag()
	case "publish":
		runPublish()
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

func runAdd() {
	if err := changeset.InteractiveAdd(); err != nil {
		fmt.Println("⚠️ Failed to add changeset:", err)
		os.Exit(1)
	}
	fmt.Println("✅ Changeset added")
	os.Exit(0)
}

func runVersion() {
	newVer, err := changeset.ApplyChangesets()
	if err != nil {
		fmt.Println("⚠️", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Version bumped to v%s\n", newVer)
	os.Exit(0)
}

func runTag() {
	version, err := changeset.GetLatestVersion()
	if err != nil {
		fmt.Println("⚠️ Failed to get latest version:", err)
		os.Exit(1)
	}

	tagName := "v" + version
	checkCmd := exec.Command("git", "tag", "--list", tagName)
	out, _ := checkCmd.Output()
	if string(out) != "" {
		fmt.Printf("⚠️ Tag %s already exists, skipping.\n", tagName)
		os.Exit(0)
	}

	createTag(tagName)

	if flagPush {
		pushTags()
	}
	os.Exit(0)
}

func runPublish() {
	nextVer, bumpType, err := changeset.CalculateNextVersion()
	if err != nil {
		fmt.Println("⚠️", err)
		os.Exit(1)
	}

	previewRelease(nextVer, bumpType)

	if !flagYes {
		confirmRelease()
	}

	tagName := bumpVersion()
	commitChanges(tagName)

	createTag(tagName)

	if flagPush {
		pushTags()
	}

	fmt.Printf("🎉 Published: %s\n", tagName)
	os.Exit(0)
}

func previewRelease(nextVer string, bumpType enums.ReleaseType) {
	fmt.Println("📦 Release preview")
	fmt.Println("------------------")
	fmt.Printf(" Pending bump : %s\n", bumpType)
	fmt.Printf(" Next version : v%s\n\n", nextVer)
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

func bumpVersion() string {
	newVer, err := changeset.ApplyChangesets()
	if err != nil {
		fmt.Println("⚠️", err)
		os.Exit(1)
	}
	tagName := "v" + newVer
	fmt.Printf("✅ Version bumped: %s\n", tagName)
	return tagName
}

func commitChanges(tagName string) {
	if err := runCmd("git", "add", "-A"); err != nil {
		fmt.Println("⚠️ No changes to commit.")
	} else if err := runCmd("git", "commit", "-m", fmt.Sprintf("chore 🚀: release %s", tagName)); err != nil {
		fmt.Println("⚠️ No changes to commit.")
	} else {
		fmt.Printf("✅ Committed release changes: chore 🚀: release %s\n", tagName)
	}
}

func createTag(tagName string) {
	message := getChangelogForTag(tagName)
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

func getChangelogForTag(tagName string) string {
	// tagName is "v1.2.3", version is "1.2.3"
	version := strings.TrimPrefix(tagName, "v")
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
	fmt.Println("  publish     Bump version, commit, and create a tag")
	fmt.Println("  help        Show this help message")
	fmt.Println("  --version, -v    Show changesetgoo CLI version")
	fmt.Println("\nFlags:")
	fmt.Println("  --yes            Auto-confirm publish without prompting")
	fmt.Println("  --push           Auto-push commits and tags after publish")
}

func printCLIVersion() {
	// Read build information embedded in the binary
	if info, ok := debug.ReadBuildInfo(); ok {
		// If compiled via 'go install @v1.2.3', this will be populated
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			fmt.Printf("CLI Tool Version: %s\n", info.Main.Version)
			return
		}

		// Fallback: Look for the local Git commit hash embedded during 'go build'
		var revision string
		var vcsTime string
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				revision = setting.Value
			}
			if setting.Key == "vcs.time" {
				vcsTime = setting.Value
			}
		}

		if revision != "" {
			fmt.Printf("CLI Tool Local Build (Commit: %s, Built: %s)\n", revision[:7], vcsTime)
			return
		}
	}
	fmt.Println("changesetgoo: invicible")
}
