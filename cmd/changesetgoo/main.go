package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ChanduBobbili/changesetgoo/changeset"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: changeset [add|version|tag|publish]")
		return
	}

	switch os.Args[1] {
	case "add":
		if err := changeset.InteractiveAdd(); err != nil {
			fmt.Println("⚠️ Failed to add changeset")
		}

	case "version":
		newVer, err := changeset.ApplyChangesets()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("✅ Version bumped to v%s\n", newVer)

	case "tag":
		version, err := changeset.GetLatestVersion()
		if err != nil {
			fmt.Println("⚠️ Failed to get latest version")
			return
		}

		tagName := "v" + version

		checkCmd := exec.Command("git", "tag", "--list", tagName)
		out, _ := checkCmd.Output()
		if string(out) != "" {
			fmt.Printf("⚠️ Tag %s already exists, skipping.\n", tagName)
			return
		}

		tagCmd := exec.Command("git", "tag", tagName)
		tagCmd.Stdout = os.Stdout
		tagCmd.Stderr = os.Stderr
		if err := tagCmd.Run(); err != nil {
			panic(err)
		}
		fmt.Printf("✅ Git tag %s created\n", tagName)

	case "publish":
		// Preview next version
		nextVer, bumpType, err := changeset.CalculateNextVersion()
		if err != nil {
			fmt.Println("⚠️ ", err)
			return
		}

		fmt.Println("📦 Release preview")
		fmt.Println("------------------")
		fmt.Printf(" Pending bump : %s\n", bumpType)
		fmt.Printf(" Next version : v%s\n\n", nextVer)

		fmt.Print("Do you want to continue with this release? (y/n): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("❌ Publish cancelled.")
			return
		}

		// Apply changesets
		newVer, err := changeset.ApplyChangesets()
		if err != nil {
			fmt.Println("⚠️ ", err)
			return
		}
		tagName := "v" + newVer
		fmt.Printf("✅ Version bumped: %s\n", tagName)

		// Commit all changes
		addCmd := exec.Command("git", "add", "-A")
		addCmd.Stdout = os.Stdout
		addCmd.Stderr = os.Stderr
		if err := addCmd.Run(); err != nil {
			fmt.Printf("⚠️ No changes to commit (skipping commit).")
			return
		}

		commitMsg := fmt.Sprintf("chore 🚀: release %s", tagName)
		commitCmd := exec.Command("git", "commit", "-m", commitMsg)
		commitCmd.Stdout = os.Stdout
		commitCmd.Stderr = os.Stderr
		if err := commitCmd.Run(); err != nil {
			fmt.Println("⚠️ No changes to commit (skipping commit).")
		} else {
			fmt.Printf("✅ Committed release changes: %s\n", commitMsg)
		}

		// Create git tag
		checkCmd := exec.Command("git", "tag", "--list", tagName)
		out, _ := checkCmd.Output()
		if string(out) != "" {
			fmt.Printf("⚠️ Tag %s already exists, skipping.\n", tagName)
			return
		}

		tagCmd := exec.Command("git", "tag", tagName)
		tagCmd.Stdout = os.Stdout
		tagCmd.Stderr = os.Stderr
		if err := tagCmd.Run(); err != nil {
			fmt.Println("⚠️ ", err)
			return
		}

		fmt.Printf("🎉 Published: %s (all changes committed, tag created, not pushed)\n", tagName)
	}
}
