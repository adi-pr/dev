# dev

A personal CLI for my Linux workstation.

`dev` exists to remove repetitive friction from my development workflow and provide one consistent interface for tasks I perform often.

The goal is not to replace existing tools.

The goal is to build:

> **The interface I wish my workstation had.**

## Principles

* **Personal first** — built around my workflow.
* **Automate repetition** — repeated command sequences can become one useful command.
* **Use existing tools** — build on Git, SSH, Linux utilities, language tooling, and package managers.
* **One interface** — related workflows live under `dev` instead of scattered scripts.
* **Structured output** — commands should support both human-readable output and formats such as JSON.

## Desktop integration

`dev` will eventually power features in [my shell customisations](https://github.com/adi-pr/shell/tree/custom), my fork of the [Caelestia](https://github.com/caelestia-dots/shell) rice for Arch Linux.

This is why commands offer JSON output: the shell can call `dev` and read its results directly, so the terminal and the desktop share the same logic.

## Commands

The CLI may eventually include areas such as:

```text
dev
├── project
├── git
├── environment
├── ssh
├── system
├── machine
└── homelab
```

These are not a checklist. Features should be added only when they solve a real workflow.

Examples:

```bash
dev project list
dev project open memocore

dev git dirty
dev git recent

dev environment doctor

dev homelab status

dev system status
dev system status --json
```

## Architecture

```text
CLI commands
     ↓
domain logic
     ↓
Linux / external tools
```

Cobra commands should stay thin. Project logic should live outside the CLI layer.

User-specific configuration belongs in:

```text
~/.config/dev/
```

## What `dev` is not

`dev` is not:

* a replacement for Linux
* a replacement for Git
* a replacement for SSH
* a package manager
* a giant collection of aliases
* a framework for hypothetical users
* a feature checklist

## Feature rule

Before adding something, ask:

> **Is this something I repeatedly do that would genuinely be better as part of my personal development environment?**

If yes, build it.

If not, use the existing tool.

## Development

```bash
go run .
go test ./...
go vet ./...
go build .
go install .
```
