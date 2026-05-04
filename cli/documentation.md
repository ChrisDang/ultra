# Vibe Cloud

## What is Vibe Cloud?

Vibe Cloud is a CLI that wraps the Vercel, Supabase, and Expo CLIs into a single interface. It detects your project's stack, links the right providers, and deploys everything in one command.

It's built to work with Claude. Every command outputs structured JSON with a `claude_instructions` field, so Claude can read the result and decide what to do next without parsing human-readable text.

## Why?

Deploying an app today means juggling multiple CLIs, each with its own auth, configuration, and deployment process. If you built an app with an AI coding assistant, this is usually where you get stuck.

Vibe Cloud solves this by:

- **Detecting your stack automatically** — `vibecloud init` scans your project for framework indicators (Next.js, React, Expo, Go, Python, static HTML, etc.) and determines which providers to use.
- **Deploying everything in one command** — `vibecloud deploy` runs Supabase migrations, deploys Vercel frontends, and triggers Expo builds in the correct dependency order.
- **Being AI-native** — All output is structured JSON designed for LLM consumption. Claude doesn't need to parse terminal output; it reads JSON with explicit next-step instructions.
- **Handling auth once** — `vibecloud login` authenticates with all detected providers in a single pass and persists the status locally.

## Who is this for?

1. **Non-technical builders** who created an app with an AI assistant and need to get it live without learning DevOps.
2. **Developers who don't want to do DevOps** — they know what Vercel and Supabase are, but don't want to wire them together for every project.
3. **AI agents** — Claude and other AI coding tools that can write code but need a deployment interface.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| CLI | Go, Cobra |
| Providers | Vercel (frontend hosting), Supabase (database/edge functions), Expo/EAS (mobile builds) |

## Commands

### `vibecloud init`

Scans the current directory, detects the framework and providers, writes a `.vibecloud.json` config file, and links detected providers (runs `vercel link`, `supabase link`, etc.).

**Detection logic:**
- `next.config.*` → Next.js → Vercel
- `expo` dependency in package.json → Expo
- `react` dependency → React → Vercel
- `Dockerfile`, `requirements.txt`, `go.mod` → Supabase
- `index.html` → static site → Vercel
- `supabase/` directory → adds Supabase provider

### `vibecloud deploy`

Deploys to all detected providers in dependency order: Supabase (migrations + edge functions) → Vercel → Expo (EAS build).

Flags:
- `--provider <name>` — deploy only a specific provider
- `--prod` — deploy to production (Vercel)

### `vibecloud status`

Shows project status from each linked provider by running the appropriate CLI commands (`vercel ls`, `supabase projects list`, `eas build:list`).

### `vibecloud logs`

Fetches logs from a provider. Defaults to the first detected provider.

Flags:
- `--provider <name>` — which provider to fetch logs from
- `--since <duration>` — show logs since a time (e.g. `1h`, `30m`)

### `vibecloud login`

Authenticates with all provider CLIs (or a specific one with `--provider`). Saves auth status to `~/.vibecloud/config.json`.

### `vibecloud version`

Prints the CLI version.

## Output Format

All commands output JSON to stdout in this shape:

```json
{
  "success": true,
  "message": "Deployment complete",
  "data": { ... },
  "claude_instructions": "All providers deployed successfully."
}
```

Human-readable progress goes to stderr. Structured results go to stdout. This lets Claude read the JSON while the user sees progress output in their terminal.

## Getting Started

1. Install the provider CLIs you need: [Vercel CLI](https://vercel.com/docs/cli), [Supabase CLI](https://supabase.com/docs/guides/cli), [EAS CLI](https://docs.expo.dev/build/setup/)
2. Build the Vibe Cloud CLI: `cd cli && go build -o vibecloud .`
3. Authenticate: `vibecloud login`
4. Initialize your project: `vibecloud init`
5. Deploy: `vibecloud deploy`
