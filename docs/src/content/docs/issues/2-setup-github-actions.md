---
tags: ["chore"]
title: "2-setup-github-actions"
type: chore
---

# Setup GitHub Actions

## Implementation Plan

1. **Workflow Setup (`.github/workflows/ci.yml`):**
   - Create a main CI workflow triggered on push to `main` and on pull requests.
   - **Steps:** Checkout code, set up Go environment, resolve dependencies.
   - Add a linting step (e.g., `golangci-lint run`).
   - Add testing steps for unit, integration (`go test -tags=integration ./...`), and E2E tests.
   - Include a commented-out step for code coverage upload (pending Codecov setup).

2. **Release Automation (`.github/workflows/release.yml` and `.goreleaser.yaml`):**
   - Create a release workflow triggered on new tags (e.g., `v*`).
   - Initialize and configure `.goreleaser.yaml` using its free-tier features.
   - **Build Targets:** Configure matrix builds for Linux, Windows, macOS across `amd64` and `arm64` architectures.
   - **Package Managers:** Add definitions for:
     - Homebrew (using a dedicated tap).
     - Chocolatey.
     - Linux packages (rpm, deb, snap, flatpak, appimage) utilizing `nfpms`.
   - **Docker:** Prepare the Docker definition block in `.goreleaser.yaml` but leave it commented out until the Docker Hub account is configured.

3. **Announcements:**
   - Utilize GoReleaser's `announce` functionality (or a GitHub action step) to automatically broadcast new releases to platforms like Twitter, Discord, or Slack.