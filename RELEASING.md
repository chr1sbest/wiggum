# Releasing Ralph

This document describes the release process for Ralph (Wiggum).

## Versioning Policy

Ralph follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** (v2.0.0): Breaking changes to CLI interface, configuration format, or PRD schema
- **MINOR** (v1.1.0): New features, commands, or backward-compatible functionality
- **PATCH** (v1.0.1): Bug fixes, documentation updates, or internal improvements

Examples:
- Adding `ralph pr` command → MINOR version bump
- Fixing a crash in the loop → PATCH version bump
- Changing `.ralph/prd.json` schema in an incompatible way → MAJOR version bump

## Release Process

### 1. Pre-release Checklist

Before cutting a release, verify:

- [ ] All tests pass: `make test`
- [ ] Code is formatted: `make fmt`
- [ ] CI is green on the main branch
- [ ] CHANGELOG.md is updated (if maintained)
- [ ] README.md reflects current functionality
- [ ] Version numbers in documentation are up to date

### 2. Creating a Release

Releases are automated via GitHub Actions when you push a version tag:

```bash
# Ensure you're on main and up to date
git checkout main
git pull origin main

# Create and push a version tag
git tag v1.2.3
git push origin v1.2.3
```

**Tag format:** Tags must follow the pattern `v*` (e.g., `v1.0.0`, `v1.2.3`)

### 3. What Happens Automatically

When you push a version tag, the `.github/workflows/release.yml` workflow:

1. Runs tests (`go test ./...`)
2. Builds binaries for multiple platforms:
   - Darwin (macOS) - amd64 and arm64
   - Linux - amd64 and arm64
3. Creates platform-specific archives (`.tar.gz`)
4. Creates a GitHub Release at `https://github.com/chr1sbest/wiggum/releases/tag/v1.2.3`
5. Uploads build artifacts as release assets

Build artifacts are compiled with:
- Version information embedded via ldflags
- Stripped symbols (`-s -w`) for smaller binaries
- Trimmed paths for reproducible builds

### 4. Distribution Channels

#### Homebrew Tap

Ralph is distributed via a Homebrew tap at `chr1sbest/tap`.

**Current process:** Manual update required. After creating a GitHub release:

1. Navigate to the Homebrew tap repository
2. Update the formula with the new version and SHA256 checksums
3. Test the formula locally: `brew install --build-from-source ./ralph.rb`
4. Push the updated formula

**Future automation:** Consider using [goreleaser](https://goreleaser.com/) to automatically update the Homebrew formula when a release is created.

#### Go Install

Users can install directly from source:

```bash
go install github.com/chr1sbest/wiggum/cmd/ralph@latest
```

This pulls from the latest commit on main. Version tags are available via:

```bash
go install github.com/chr1sbest/wiggum/cmd/ralph@v1.2.3
```

### 5. Post-release Steps

After a release is published:

1. Verify the GitHub Release page looks correct
2. Test installation via Homebrew (after tap update)
3. Test `go install` installation
4. Announce the release (if applicable):
   - GitHub Discussions
   - Project README shields/badges
   - Social media or community channels

### 6. Versioning in Code

Version information is embedded at build time via ldflags in `.github/workflows/release.yml`:

```bash
-ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"
```

Users can check their version with:

```bash
ralph version
```

Local development builds (via `go install ./cmd/ralph`) may show version `dev` since build-time variables aren't set.

## Hotfix Process

For urgent fixes:

1. Create a hotfix branch from the release tag:
   ```bash
   git checkout -b hotfix/v1.2.4 v1.2.3
   ```
2. Apply the fix and commit
3. Merge the hotfix to main:
   ```bash
   git checkout main
   git merge hotfix/v1.2.4
   ```
4. Tag the hotfix version:
   ```bash
   git tag v1.2.4
   git push origin v1.2.4
   ```
5. Follow the normal release automation

## Rollback

If a release has critical issues:

1. **DO NOT** delete the Git tag or GitHub release (breaks user installations)
2. Instead, release a new version with the fix (either revert commits or fix forward)
3. Document the issue in release notes
4. Update documentation to recommend the newer version

## Release Schedule

Ralph does not follow a fixed release schedule. Releases are cut when:

- A meaningful new feature is complete
- Critical bugs are fixed
- A batch of quality-of-life improvements have accumulated

Aim for roughly:
- PATCH releases: As needed for urgent fixes
- MINOR releases: Every 4-8 weeks for new features
- MAJOR releases: Only when necessary for breaking changes

## Questions?

For questions about the release process, open an issue or discussion on GitHub.
