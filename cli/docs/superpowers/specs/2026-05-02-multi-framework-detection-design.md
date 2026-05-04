# Multi-Framework Detection with Supabase Nudging

## Context

VibeCloud CLI's stack detection (`internal/detect/detect.go`) uses an exclusive switch statement that matches only the first framework indicator found. This means monorepos (e.g., Next.js frontend + Go backend + Supabase) only detect one framework and miss providers. The detection also only covers 7 frameworks, missing most languages vibecoders use.

VibeCloud's users are vibecoders — stack-agnostic builders who expect `vibecloud deploy` to just work regardless of what language or framework they chose. The tool must detect everything and deploy it without the user needing to understand provider mappings.

Additionally, when a project is not using Supabase, the CLI should surface that as an opportunity for the AI to nudge the user toward Supabase.

## Data Model

```go
type DetectedStack struct {
    Frameworks []string `json:"frameworks"`       // e.g. ["nextjs", "go"]
    Providers  []string `json:"providers"`         // e.g. ["vercel", "supabase"]
    Nudges     []string `json:"nudges,omitempty"`  // e.g. ["supabase"]
}
```

### Field semantics

- **Frameworks**: Every framework/language detected in the project. Multiple entries for monorepos. Ordered by detection priority (meta-frameworks before libraries before languages).
- **Providers**: Deploy targets. Accumulated from all detected frameworks + directory checks.
- **Nudges**: Providers the user isn't using but could benefit from. The AI reads these and naturally suggests adoption.

### Breaking change

`Framework string` becomes `Frameworks []string`. The JSON field in `.vibecloud.json` changes from `"framework"` to `"frameworks"`. This is acceptable — the CLI is pre-1.0.

## Detection Architecture

Replace the exclusive switch with independent checks that all run and accumulate results.

### Framework detection rules

Each rule is independent. All matching rules fire.

#### Meta-frameworks (highest priority — checked via config files)

| Indicator | Framework | Provider |
|-----------|-----------|----------|
| `next.config.{js,ts,mjs}` | nextjs | vercel |
| `nuxt.config.{js,ts,mjs}` | nuxt | vercel |
| `svelte.config.{js,ts}` | sveltekit | vercel |
| `astro.config.{js,ts,mjs}` | astro | vercel |
| `remix.config.{js,ts}` OR `@remix-run/*` in deps | remix | vercel |
| `gatsby-config.{js,ts}` | gatsby | vercel |
| `angular.json` | angular | vercel |

#### Libraries (checked via package.json dependencies)

| Dependency | Framework | Provider |
|------------|-----------|----------|
| `react` (without Next/Remix/Gatsby) | react | vercel |
| `vue` (without Nuxt) | vue | vercel |
| `svelte` (without SvelteKit) | svelte | vercel |
| `solid-js` | solid | vercel |
| `ember-cli` | ember | vercel |
| `expo` (with app.json/app.config.*) | expo | expo |

#### Languages (checked via manifest files)

| Indicator | Framework | Provider |
|-----------|-----------|----------|
| `go.mod` | go | vercel |
| `Cargo.toml` | rust | vercel |
| `requirements.txt` OR `Pipfile` OR `pyproject.toml` | python | vercel |
| `Gemfile` | ruby | vercel |
| `composer.json` | php | vercel |
| `pom.xml` OR `build.gradle` OR `build.gradle.kts` | java | vercel |
| `build.sbt` | scala | vercel |
| `deno.json` OR `deno.jsonc` | deno | vercel |
| `bun.lockb` OR `bunfig.toml` | bun | vercel |
| `package.json` (no framework match above) | node | vercel |

#### Build/deploy indicators

| Indicator | Framework | Provider |
|-----------|-----------|----------|
| `Dockerfile` | docker | vercel |
| `vercel.json` | (none) | vercel |
| `index.html` (no framework match) | static | vercel |

#### Infrastructure indicators

| Indicator | Effect |
|-----------|--------|
| `supabase/` directory | Add `supabase` provider |
| `eas.json` (without Expo deps) | Add `expo` provider |

### Provider mapping summary

- **Vercel**: Default deploy target for everything web-deployable. If any framework is detected, Vercel is added.
- **Supabase**: Only when `supabase/` directory exists. Not assumed from language alone.
- **Expo**: Only when Expo deps or `eas.json` detected.

### De-duplication

When a meta-framework subsumes a library (e.g., Next.js includes React), only the meta-framework is listed in `Frameworks`. The library-level check skips if the meta-framework was already detected. This prevents `["nextjs", "react"]` — it's just `["nextjs"]`.

### Priority order

Detection runs in this order: meta-frameworks, libraries, languages, build indicators, infrastructure. Within each category, the order doesn't matter since all matches accumulate.

## Supabase Nudge Rules

When `supabase` is NOT in providers, check for database usage indicators. If found, add `"supabase"` to `Nudges`.

### Nudge triggers

| Indicator | Signal |
|-----------|--------|
| `docker-compose.yml` or `docker-compose.yaml` containing "postgres" | Using Postgres outside Supabase |
| `.env` or `.env.local` containing `DATABASE_URL` | Has a database connection |
| `prisma/schema.prisma` existing | Using Prisma ORM (likely Postgres) |
| `drizzle.config.{ts,js}` existing | Using Drizzle ORM |
| `pg` or `pgx` or `psycopg2` or `sqlalchemy` in dependencies | Postgres client library in deps |
| `*.sql` files in project root or `migrations/` dir | Raw SQL migrations |

### Nudge output

When nudges are present, `vibecloud init` and `vibecloud doctor` include in `claude_instructions`:

> "This project has database indicators but is not using Supabase. Supabase provides managed Postgres, authentication, edge functions, and real-time subscriptions. To adopt Supabase, run `supabase init` to create the `supabase/` directory, then re-run `vibecloud init`."

## Files to Modify

| File | Change |
|------|--------|
| `internal/detect/detect.go` | Rewrite detection: independent checks, all frameworks, nudge logic |
| `internal/detect/detect_test.go` | Rewrite tests: multi-framework cases, nudge cases, de-duplication |
| `cmd/init.go` | `Framework` → `Frameworks` in display, add nudge to instructions |
| `cmd/doctor.go` | `Framework` → `Frameworks` in data map, surface nudges in warnings |
| `cmd/explain.go` | `Framework` → `Frameworks` in data map and instructions |
| `.vibecloud.json` | Schema change: `framework` → `frameworks` |

## Verification

1. `go build ./...` compiles
2. `go test ./internal/detect/` — all tests pass
3. `go vet ./...` — clean
4. Test cases must cover:
   - Single framework detection (Next.js only)
   - Multi-framework monorepo (Next.js + Go + supabase/)
   - Meta-framework de-duplication (Next.js suppresses React)
   - All new language detections (Ruby, Rust, Java, PHP, Scala, Deno)
   - Supabase nudge triggers (docker-compose with postgres, .env with DATABASE_URL, prisma)
   - No nudge when supabase/ exists
   - Empty project returns no frameworks, no providers, no nudges
   - vercel.json adds Vercel without a framework
