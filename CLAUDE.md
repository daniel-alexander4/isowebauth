# CLAUDE.md

## Project
isowebauth - Desktop app (Wails v2 + Go) that signs web authentication challenges using local SSH keys.

## Build & Run
- `go build ./...` to verify compilation
- `wails dev` to run in dev mode
- Tests: `go test ./internal/... -count=1`

## CI
- GitHub Actions workflow: `.github/workflows/release.yml`
- Linux build pinned to `ubuntu-22.04` for `libwebkit2gtk-4.0-dev` compatibility

## Architecture Notes
- Auth/consent is only requested for signing operations, not for origin validation. If origins are configured correctly, requests from those origins proceed without additional auth prompts. Do not add auth gates to origin checks.
