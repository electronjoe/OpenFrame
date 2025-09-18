# Plan to Remove Chronological Photo Order Support

1. Simplify slideshow ordering in `cmd/openframe/main.go` by deleting the conditional branch that sorts photos chronologically. Always shuffle the loaded photos before building slides, remove the `sort` import, and keep the time-based seeding so each run is randomized.
2. Update `internal/config/config.go` to drop the `Randomize` field from the `Config` struct. Confirm `Read()` still succeeds when legacy configs contain an unused `randomize` key (the JSON decoder already ignores unknown fields, but we should note this in comments or docs).
3. Adjust documentation to reflect the new invariant: revise `REQUIREMENTS.md` and `DESIGNDOC.md` to describe random order playback instead of chronological sequencing, and remove any setup steps or diagrams that mention a `randomize` flag. While there, trim references to “skip by year” timelines that assumed chronological order.
4. Refresh contributor-facing notes in `AGENTS.md` (and any other guides) so they no longer instruct maintainers to document new config fields for ordering, and instead call out that random order is the only supported mode.
5. Sweep the repository for lingering terms like “chronological order” or “randomize” to ensure we eliminate stale comments, error messages, or roadmap items. Update any logging or TODOs that still imply both modes exist.
6. Run `go fmt ./...`, `go vet ./...`, and `go test ./...` to verify the codebase stays healthy after the cleanup.
