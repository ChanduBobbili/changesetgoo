## 2.0.0

### Major Changes

- Add configuration support for release workflows, including configurable changeset paths, tag prefixes, issue formatting, changelog settings, and publish-time commit behavior.
- Introduced `changedFilePattern` to allow filtering of changed files against user-defined patterns, ensuring version bumps are only triggered by relevant file modifications.

### Minor Changes

- Migrated internal Git operations from `os/exec` shell commands to the `go-git` library to make the CLI fully self-contained and cross-platform compatible.

### Patch Changes

- Migrate description input to `chzyer/readline` to fix multi-print pasting bugs, add interrupt handling for clean exits, and resolve filepath variable shadowing.

## 1.2.1

### Patch Changes

- Fixed the changesetgoo version

## 1.2.0

### Minor Changes

- Added changelog as Git Tag Message

## 1.1.1

### Patch Changes

- Added flag to show the installed version

## 1.1.0

### Minor Changes

- Added the flag for auto-confirm, push, custom file path for CHANGELG.md
- Checked the gocyclo report && improved the code.

## 1.0.0

### Major Changes

- Added add, version, tag, publish flags for respective stage
- Generating && Updating the CHANGELOG.md file

