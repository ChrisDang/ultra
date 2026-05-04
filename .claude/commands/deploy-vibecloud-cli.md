Release a new version of the VibeCloud CLI.

Walk the user through this checklist step by step. Verify each step before moving to the next.

## Pre-release

1. Ensure all changes are committed and pushed to `main`.
2. Run tests: `cd cli && make test`
3. Run lint: `cd cli && make lint`
4. Verify a local build works: `cd cli && make build && ./vibecloud version`

## Release

5. Ask the user what version to tag (e.g. `v0.2.0`). Suggest the next version based on `git tag --list 'v*' --sort=-v:refname | head -5`.
6. Create and push the tag:
   ```
   git tag <version>
   git push origin <version>
   ```
7. Watch the workflow run to completion:
   ```
   gh run list --repo ChrisDang/vibecloud --limit 1
   gh run watch <run-id> --repo ChrisDang/vibecloud
   ```
   If it fails, pull logs with `gh run view <run-id> --repo ChrisDang/vibecloud --log` and fix locally.

## Post-release verification

8. Verify the release exists in **both** repos (private builds, public distributes):
   ```
   gh release view <version> --repo ChrisDang/vibecloud
   gh release view <version> --repo ChrisDang/vibecloud-releases
   ```
9. Verify the install script fetches the new version:
   ```
   curl -fsSL https://raw.githubusercontent.com/ChrisDang/vibecloud-releases/main/install.sh | sh
   vibecloud version
   ```
10. Report the public release URL to the user: `https://github.com/ChrisDang/vibecloud-releases/releases/tag/<version>`

## Reminders

- The private repo (`ChrisDang/vibecloud`) builds binaries via GoReleaser and publishes them to the public repo (`ChrisDang/vibecloud-releases`) using the `RELEASE_PAT` secret.
- Release builds use garble for obfuscation and `-s -w` to strip symbols (configured in `.goreleaser.yml`).
- Users install/upgrade by re-running: `curl -fsSL https://raw.githubusercontent.com/ChrisDang/vibecloud-releases/main/install.sh | sh`
- To pin a version: `VIBECLOUD_VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/ChrisDang/vibecloud-releases/main/install.sh | sh`
- If the install script in the public repo needs updating, push changes to both repos.
