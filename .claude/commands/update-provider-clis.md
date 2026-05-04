Detect version drift between vibecloud and its provider CLIs (Vercel, Supabase, EAS), then open a PR to fix it.

## Context

vibecloud wraps three provider CLIs. When they ship breaking changes, vibecloud can silently break. The file `cli/provider_versions.json` tracks the tested version range for each provider. The Go code in `cli/internal/version/providers.go` reads this manifest at runtime to warn users about drift via `vibecloud doctor`.

## Steps

### 1. Fetch latest published versions

Use WebSearch to find the current latest stable version for each provider CLI:

- **Vercel CLI**: search `"vercel CLI latest version npm 2026"`
- **Supabase CLI**: search `"supabase CLI latest version github 2026"`
- **EAS CLI**: search `"eas-cli latest version npm 2026"`

For each, extract the latest stable semver (e.g. `51.8.0`).

### 2. Read the current manifest

Read `cli/provider_versions.json`. Compare each provider's `max_tested_version` against the latest published version you found.

If all providers are up to date (latest <= max_tested_version), report "No drift detected" and stop.

### 3. Check for breaking changes

For each provider where drift is detected (latest > max_tested_version), use WebSearch and WebFetch to check for breaking changes between the tested version and latest:

- **Vercel**: fetch `https://github.com/vercel/vercel/releases` and search for breaking changes between the old and new versions
- **Supabase**: fetch `https://github.com/supabase/cli/releases` and search for breaking changes
- **EAS**: fetch the EAS CLI CHANGELOG.md from GitHub and look for breaking changes (major version bumps are frequent here — pay close attention)

Summarize what changed for each provider:
- New major versions (breaking)
- Deprecated/removed flags or commands
- Changed output formats
- New required config fields
- Auth flow changes

### 4. Update the manifest

Edit `provider_versions.json`:
- Set `tested_version` to the latest version
- Set `max_tested_version` to the latest version
- If there was a breaking major version bump, update `min_version` to the new major version's `.0.0`

### 5. Update Go code if needed

If any breaking changes affect vibecloud's usage of a provider CLI, check these files for necessary updates:

- `cli/cmd/doctor.go` — auth checks (`checkAuth`) and link checks (`checkLinked`)
- `cli/cmd/deploy.go` — deploy commands and flag usage
- `cli/cmd/login.go` — login flows
- `cli/cmd/status.go` — status commands
- `cli/cmd/logs.go` — log fetching commands
- `cli/cmd/init.go` — init/link commands
- `cli/internal/exec/install.go` — install commands

Make the minimum code changes needed to stay compatible. If no code changes are needed, skip this step.

### 6. Verify the build

Run `cd cli && go build ./...` to verify everything compiles.

### 7. Create a PR

Create a new branch, commit the changes, and open a PR:

- Branch name: `chore/update-provider-clis-YYYY-MM-DD` (use today's date)
- Commit message should summarize which providers were updated and whether breaking changes were found
- PR title: `chore: update provider CLI versions`
- PR body should include:
  - A table showing old version -> new version for each provider
  - A summary of breaking changes found (or "none")
  - Any code changes made to handle breaking changes

## Important

- Do NOT update versions for providers where no new version was published
- Do NOT make speculative code changes — only change code if a concrete breaking change requires it
- If a breaking change is too complex to handle automatically, note it in the PR description and flag it for manual review
- Always verify the build compiles before committing
