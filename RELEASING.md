# Release Guide

This document describes how to create releases for the mcpchecker project.

## Release Types

The project supports three types of releases:

1. **Stable Releases** (`vX.Y.Z`) - Production-ready releases
2. **Pre-releases** (`vX.Y.Z-rc.N`) - Release candidates for testing
3. **Nightly Releases** (`nightly`) - Automated daily builds

## Prerequisites

- Write access to the repository
- Ability to create and push tags
- Updated CHANGELOG.md

## Stable Release

Stable releases follow semantic versioning (vX.Y.Z) and are triggered by pushing a version tag.

### Steps

1. **Update CHANGELOG.md**

   Add a new section for the version you're releasing:
   ```markdown
   ## [X.Y.Z]

   ### Added
   - New feature description

   ### Changed
   - Change description

   ### Fixed
   - Bug fix description
   ```

2. **Commit the changelog**
   ```bash
   git add CHANGELOG.md
   git commit -m "docs: update changelog for vX.Y.Z"
   git push origin main
   ```

3. **Create and push the version tag**

   You can do this through the github UI (on the releases page), or by git with the following commands:
   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

4. **Automated workflow**

   The GitHub Actions workflow will:
   - Validate the tag format (must be `vX.Y.Z`)
   - Verify that CHANGELOG.md contains a section for this version
   - Run tests
   - Build binaries for all platforms (linux, darwin, windows) and architectures (amd64, arm64)
   - Sign artifacts with cosign
   - Create a GitHub release with the changelog
   - Upload signed artifacts to the release

5. **Verify the release**

   Check the [releases page](https://github.com/mcpchecker/mcpchecker/releases) to ensure:
   - The release was created successfully
   - All artifacts are present (12 zip files + 12 bundles)
   - The changelog was extracted correctly

## Pre-release

Pre-releases use the format `vX.Y.Z-rc.N` where N is the release candidate number.

### Steps

1. **Update CHANGELOG.md**

   You can either:
   - Add a section for the target version (e.g., `[1.0.0]`)
   - Use the `[Unreleased]` section (will be used if version section doesn't exist)

2. **Commit the changelog**
   ```bash
   git add CHANGELOG.md
   git commit -m "docs: update changelog for v1.0.0-rc.1"
   git push origin main
   ```

3. **Create and push the pre-release tag**
   ```bash
   git tag v1.0.0-rc.1
   git push origin v1.0.0-rc.1
   ```

4. **Automated workflow**

   The workflow will:
   - Validate the tag format (must be `vX.Y.Z-rc.N`)
   - Extract changelog from version section or fall back to Unreleased
   - Build and sign binaries
   - Create a pre-release on GitHub
   - Upload artifacts

### Testing Pre-releases

Pre-releases are marked as "pre-release" on GitHub and won't be considered the "latest" release. Use them to:
- Test release artifacts before stable release
- Get feedback from early adopters
- Verify the release process

## Nightly Release

Nightly releases are automated and run daily at 02:00 UTC. They use the tag `nightly`.

### Automated Process

The nightly workflow:
- Checks for unreleased commits since the last stable release
- Skips if no new commits exist
- Deletes and recreates the `nightly` tag and release
- Builds binaries from the main branch
- Extracts changelog from the `[Unreleased]` section

### Manual Nightly

To trigger a nightly build manually:

1. Go to the [Actions tab](https://github.com/mcpchecker/mcpchecker/actions/workflows/nightly.yaml)
2. Click "Run workflow"
3. Select the branch (usually `main`)
4. Click "Run workflow"

## Versioning Guidelines

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** (X.0.0): Incompatible API changes
- **MINOR** (x.Y.0): New functionality, backwards compatible
- **PATCH** (x.y.Z): Bug fixes, backwards compatible

### Examples

- `v1.0.0` - First stable release
- `v1.1.0` - Added new features
- `v1.1.1` - Bug fixes
- `v2.0.0` - Breaking changes
- `v1.2.0-rc.1` - First release candidate for v1.2.0
- `v1.2.0-rc.2` - Second release candidate for v1.2.0


## Security: Artifact Signing

All release artifacts are signed using [cosign](https://github.com/sigstore/cosign) v4 with keyless signing (via GitHub OIDC). Signatures and certificates are stored in bundle files for simplified verification.

### Verifying Signatures

Users can verify artifact signatures using the bundle format:

```bash
# Download artifacts
wget https://github.com/mcpchecker/mcpchecker/releases/download/v1.0.0/mcpchecker-linux-amd64.zip
wget https://github.com/mcpchecker/mcpchecker/releases/download/v1.0.0/mcpchecker-linux-amd64.zip.bundle

# Verify using bundle (simplified format)
cosign verify-blob \
  --bundle mcpchecker-linux-amd64.zip.bundle \
  --certificate-identity-regexp 'https://github.com/mcpchecker/mcpchecker' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  mcpchecker-linux-amd64.zip
```

The bundle file contains both the signature and certificate, making verification simpler compared to the older separate `.sig` and `.pem` files.
