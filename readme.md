# changesetgoo

[![Go Reference](https://pkg.go.dev/badge/github.com/ChanduBobbili/changesetgoo.svg)](https://pkg.go.dev/github.com/ChanduBobbili/changesetgoo)
[![Go Report Card](https://goreportcard.com/badge/github.com/ChanduBobbili/changesetgoo)](https://goreportcard.com/report/github.com/ChanduBobbili/changesetgoo)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

A lightweight Go-based CLI for managing **semantic versioning** and **changelogs**, inspired by [Changesets](https://github.com/changesets/changesets).

`changesetgoo` helps you track changes across releases with temporary changeset files, automatically bump versions, update your `CHANGELOG.md`, and create semantic Git tags.

---

## ✨ Features

* 📦 Add changesets (`major`, `minor`, `patch`) with descriptions.
* 📝 Merge changesets into a single `CHANGELOG.md`.
* 🔖 Auto-bump versions (`semver` rules applied).
* 🏷️ Create semantic Git tags (local only, no push).
* 🚀 Publish workflow: apply changesets, bump version, update changelog, commit, tag, and optionally push.
* ⚡ Written in Go — single binary, no Node.js required.

---

## 📦 Installation

You can install the CLI globally using `go install`:

```sh
go install github.com/ChanduBobbili/changesetgoo/cmd/changesetgoo@latest
```

This will place the `changesetgoo` binary into your `$GOPATH/bin` (make sure it’s in your `$PATH`).
Adding the path in `.bashrc` OR `.zshrc`
```sh
export GOPATH="$HOME/go"
export PATH="$GOPATH/bin:$PATH"
```

---

## ⚡ Usage

After installing, you can run:

```sh
changesetgoo <command> [flags]
```

### Commands

#### 1. Add a changeset

```sh
changesetgoo add
```

* Prompts for bump type: **major / minor / patch**.
* Asks for a description.
* Creates a temporary Markdown file under `.changeset/`.

Example:

```sh
> changesetgoo add
✔ minor
Enter change description: Added support for new API
✅ Changeset added
```

---

#### 2. Version bump & changelog update

```sh
changesetgoo version
```

* Reads all `.changeset/*.md` files.
* Calculates the **highest-priority bump** (`major` > `minor` > `patch`).
* Updates the changelog file with merged notes.
* Deletes temporary `.changeset/*.md` files.

Example:

```sh
> changesetgoo version
✅ Version bumped to v1.2.0
```

---

#### 3. Create a Git tag

```sh
changesetgoo tag
```

* Creates a semantic Git tag (`vX.Y.Z`) for the latest version.
* **Does not push** the tag (local only).

Example:

```sh
> changesetgoo tag
✅ Git tag v1.2.0 created
```

---

#### 4. Publish a release

```sh
changesetgoo publish [--yes] [--push]
```

* Previews the next version bump and pending changes.
* Confirms release (or skips confirmation with `--yes`).
* Applies changesets, bumps version, updates changelog.
* Commits changes with a release message.
* Creates a Git tag.
* Optionally pushes commits and tags with `--push`.

Example:

```sh
> changesetgoo publish --yes --push
✅ Version bumped: v1.2.0
✅ Committed release changes
✅ Git tag v1.2.0 created
🎉 Published: v1.2.0 (pushed to remote)
```

---

## 🌍 Global Flags

| Flag                 | Description                                                       |
| -------------------- | ----------------------------------------------------------------- |
| `--yes`              | Auto-confirm publish without prompting                            |
| `--push`             | Push commits and tags to remote after publish                     |

---

## 🔄 Exit Codes

| Code | Meaning                                             |
| ---- | --------------------------------------------------- |
| `0`  | Success                                             |
| `1`  | General error (unexpected failure)                  |
| `2`  | Invalid command or usage error                      |
| `3`  | Git-related error (commit/tag/push failure)         |

---

## 📝 Changelog

View the changelog:
[CHANGELOG.md](./CHANGELOG.md)

---

## 🛠 Development

Clone the repo and run locally:

```sh
git clone https://github.com/ChanduBobbili/changesetgoo.git
cd changesetgoo
go run cmd/main.go add
```

Build the binary:

```sh
go build -o changesetgoo ./cmd
./changesetgoo --help
```

---

## 📂 Project Structure

```
.
├── changeset/        # Core logic: changelog, changeset parsing, versioning
├── cmd/              # CLI entrypoint
├── constants/        # Constants used across the project
├── enums/            # Enum types for bumps
└── go.mod            # Module definition
```

---

## 🤝 Contributing

Contributions are welcome!

* Fork the repo
* Create a feature branch
* Submit a PR 🚀

Please ensure code is formatted with `go fmt`.

---

## 📜 License

This project is licensed under the [MIT License](./LICENSE).
