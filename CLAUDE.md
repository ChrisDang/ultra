@AGENTS.md


## Deployment (VibeCloud)

This project uses **VibeCloud** for deployment. Use vibecloud commands instead of vercel CLIs directly.

### Commands
- `vibecloud deploy` — deploy all providers in dependency order
- `vibecloud deploy --prod` — deploy to production
- `vibecloud deploy --provider <name>` — deploy a single provider
- `vibecloud doctor` — preflight check (run before deploy to catch issues)
- `vibecloud explain` — full project state across all providers
- `vibecloud status` — provider status
- `vibecloud logs --provider <name>` — fetch logs
- `vibecloud login` — authenticate with all providers

### Output format
All commands output JSON to stdout with this structure:
```json
{
  "success": true|false,
  "message": "...",
  "error_code": "...",
  "data": { ... },
  "recovery": { "command": "vibecloud ...", "auto_recoverable": true },
  "claude_instructions": "Actionable next step."
}
```
Parse the `claude_instructions` field for what to do next. If `recovery.auto_recoverable` is true, you can run `recovery.command` automatically.

### Providers: vercel
