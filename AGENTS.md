# Repository Guidelines

## Project Structure & Module Organization
Primary entry point lives in `cmd/openframe/main.go`, orchestrating config parsing, photo ingestion, CEC listeners, and the Ebiten slideshow. Shared packages live under `internal/` (`config`, `photo`, `slideshow`, `cec`); keep their APIs cohesive and prefer creating new sibling packages over inflating `main`. Configuration loads from `~/.openframe/config.json`—reflect new fields in both the struct tags and documentation. Utility binaries in `cmd/cectest` and `cmd/geocode` support manual HDMI-CEC and metadata experiments. Systemd units live in `linux/`.

## Build, Test, and Development Commands
- `go run ./cmd/openframe --config ~/.openframe/config.json` starts the slideshow using your local config.
- `go build -o bin/openframe ./cmd/openframe` creates a deployable binary; keep the `bin/` path out of version control.
- `go test ./...` executes package tests; run it before every push.
- `go fmt ./...` (or `gofmt -w`) normalizes formatting; pair it with `go vet ./...` when debugging subtle issues.

## Coding Style & Naming Conventions
The project targets Go 1.22, so lean on language defaults: tabs from `gofmt`, camelCase for identifiers, and exported names only when consumed across packages. Keep Ebiten rendering code responsive—split draw/update helpers rather than embedding long switch statements. For assets or samples, prefer `testdata/` folders to keep Go tooling happy.

## Testing Guidelines
Although no tests exist yet, new logic should ship with `_test.go` files alongside the production package. Favor table-driven tests and temporary directories for photo fixtures (`t.TempDir()` plus symbolic album hierarchies). When touching time-based behavior, stub clocks via small interfaces so tests can assert deterministic slideshows. Document any manual test steps in the PR description, especially around HDMI-CEC routines.

## Commit & Pull Request Guidelines
Current history uses short, sentence-style summaries (e.g., “Frame now turns off automatically at 8p, on at 6a”). Match that tone, keep subjects under ~70 characters, and reference issues with `Refs #123` when applicable. Each PR should explain intent, list validation (commands run, photos exercised), and call out config or systemd changes. Include screenshots or logs when altering rendering or CEC handling so reviewers can reason without reproducing hardware setups.

## System & Deployment Notes
Install HDMI-CEC dependencies before running (`sudo apt-get install cec-utils`). For scheduled operation, copy the `linux/*.service` and `linux/*.timer` units into your user systemd directory, run `systemctl --user daemon-reload`, then enable the start/stop timers. Keep those instructions in sync whenever CLI flags or config expectations change.
