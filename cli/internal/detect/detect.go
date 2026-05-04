package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DetectedStack holds the result of scanning a project directory.
type DetectedStack struct {
	Frameworks []string `json:"frameworks"`
	Providers  []string `json:"providers"`
	Nudges     []string `json:"nudges,omitempty"`
}

// DetectStack scans dir for framework indicators and returns a DetectedStack.
// All checks are independent and accumulative — monorepos get all frameworks detected.
func DetectStack(dir string) DetectedStack {
	stack := DetectedStack{}

	// Track which meta-frameworks are found so we can suppress their base libraries.
	var metaFrameworks []string

	// --- Meta-frameworks (config file detection) ---

	if fileExists(dir, "next.config.js") || fileExists(dir, "next.config.ts") || fileExists(dir, "next.config.mjs") {
		stack.Frameworks = appendUnique(stack.Frameworks, "nextjs")
		metaFrameworks = append(metaFrameworks, "nextjs")
	}
	if fileExists(dir, "nuxt.config.js") || fileExists(dir, "nuxt.config.ts") || fileExists(dir, "nuxt.config.mjs") {
		stack.Frameworks = appendUnique(stack.Frameworks, "nuxt")
		metaFrameworks = append(metaFrameworks, "nuxt")
	}
	if fileExists(dir, "svelte.config.js") || fileExists(dir, "svelte.config.ts") {
		stack.Frameworks = appendUnique(stack.Frameworks, "sveltekit")
		metaFrameworks = append(metaFrameworks, "sveltekit")
	}
	if fileExists(dir, "astro.config.js") || fileExists(dir, "astro.config.ts") || fileExists(dir, "astro.config.mjs") {
		stack.Frameworks = appendUnique(stack.Frameworks, "astro")
		metaFrameworks = append(metaFrameworks, "astro")
	}
	if fileExists(dir, "remix.config.js") || fileExists(dir, "remix.config.ts") || hasAnyDep(dir, "@remix-run/node", "@remix-run/react", "@remix-run/serve") {
		stack.Frameworks = appendUnique(stack.Frameworks, "remix")
		metaFrameworks = append(metaFrameworks, "remix")
	}
	if fileExists(dir, "gatsby-config.js") || fileExists(dir, "gatsby-config.ts") {
		stack.Frameworks = appendUnique(stack.Frameworks, "gatsby")
		metaFrameworks = append(metaFrameworks, "gatsby")
	}
	if fileExists(dir, "angular.json") {
		stack.Frameworks = appendUnique(stack.Frameworks, "angular")
		metaFrameworks = append(metaFrameworks, "angular")
	}

	// --- Libraries (only if not subsumed by a meta-framework) ---

	reactSubsumed := containsAny(metaFrameworks, "nextjs", "remix", "gatsby")
	vueSubsumed := containsAny(metaFrameworks, "nuxt")
	svelteSubsumed := containsAny(metaFrameworks, "sveltekit")

	if !reactSubsumed && hasDep(dir, "react") {
		stack.Frameworks = appendUnique(stack.Frameworks, "react")
	}
	if !vueSubsumed && hasDep(dir, "vue") {
		stack.Frameworks = appendUnique(stack.Frameworks, "vue")
	}
	if !svelteSubsumed && hasDep(dir, "svelte") {
		stack.Frameworks = appendUnique(stack.Frameworks, "svelte")
	}
	if hasDep(dir, "solid-js") {
		stack.Frameworks = appendUnique(stack.Frameworks, "solid")
	}
	if hasDep(dir, "ember-cli") {
		stack.Frameworks = appendUnique(stack.Frameworks, "ember")
	}

	// Expo — needs expo dep AND an app config file.
	hasExpoConfig := fileExists(dir, "app.json") || fileExists(dir, "app.config.js") || fileExists(dir, "app.config.ts")
	if hasExpoConfig && hasDep(dir, "expo") {
		stack.Frameworks = appendUnique(stack.Frameworks, "expo")
		stack.Providers = appendUnique(stack.Providers, "expo")
	}

	// --- Languages (manifest file detection) ---

	if fileExists(dir, "go.mod") {
		stack.Frameworks = appendUnique(stack.Frameworks, "go")
	}
	if fileExists(dir, "Cargo.toml") {
		stack.Frameworks = appendUnique(stack.Frameworks, "rust")
	}
	if fileExists(dir, "requirements.txt") || fileExists(dir, "Pipfile") || fileExists(dir, "pyproject.toml") {
		stack.Frameworks = appendUnique(stack.Frameworks, "python")
	}
	if fileExists(dir, "Gemfile") {
		stack.Frameworks = appendUnique(stack.Frameworks, "ruby")
	}
	if fileExists(dir, "composer.json") {
		stack.Frameworks = appendUnique(stack.Frameworks, "php")
	}
	if fileExists(dir, "pom.xml") || fileExists(dir, "build.gradle") || fileExists(dir, "build.gradle.kts") {
		stack.Frameworks = appendUnique(stack.Frameworks, "java")
	}
	if fileExists(dir, "build.sbt") {
		stack.Frameworks = appendUnique(stack.Frameworks, "scala")
	}
	if fileExists(dir, "deno.json") || fileExists(dir, "deno.jsonc") {
		stack.Frameworks = appendUnique(stack.Frameworks, "deno")
	}
	if fileExists(dir, "bunfig.toml") || fileExists(dir, "bun.lockb") {
		stack.Frameworks = appendUnique(stack.Frameworks, "bun")
	}

	// Node fallback — only if package.json exists and no JS framework was detected above.
	hasJSFramework := containsAny(stack.Frameworks, "nextjs", "nuxt", "sveltekit", "astro", "remix", "gatsby", "angular", "react", "vue", "svelte", "solid", "ember", "expo", "deno", "bun")
	if fileExists(dir, "package.json") && !hasJSFramework {
		stack.Frameworks = appendUnique(stack.Frameworks, "node")
	}

	// --- Build/deploy indicators ---

	if fileExists(dir, "Dockerfile") {
		stack.Frameworks = appendUnique(stack.Frameworks, "docker")
	}
	if fileExists(dir, "vercel.json") {
		stack.Providers = appendUnique(stack.Providers, "vercel")
	}

	// Static HTML fallback — only if no other framework detected.
	if fileExists(dir, "index.html") && len(stack.Frameworks) == 0 {
		stack.Frameworks = appendUnique(stack.Frameworks, "static")
	}

	// --- Provider accumulation ---

	// Any detected framework means Vercel is the deploy target.
	if len(stack.Frameworks) > 0 {
		stack.Providers = appendUnique(stack.Providers, "vercel")
	}

	// Infrastructure indicators.
	if dirExists(dir, "supabase") {
		stack.Providers = appendUnique(stack.Providers, "supabase")
	}
	if fileExists(dir, "eas.json") {
		stack.Providers = appendUnique(stack.Providers, "expo")
	}

	// --- Supabase nudge ---
	if !contains(stack.Providers, "supabase") {
		stack.Nudges = detectSupabaseNudges(dir)
	}

	return stack
}

// detectSupabaseNudges checks for database usage indicators that suggest
// the project could benefit from Supabase.
func detectSupabaseNudges(dir string) []string {
	var nudges []string

	// Docker Compose with postgres.
	if fileContains(dir, "docker-compose.yml", "postgres") || fileContains(dir, "docker-compose.yaml", "postgres") {
		nudges = appendUnique(nudges, "supabase")
	}

	// .env with DATABASE_URL.
	if fileContains(dir, ".env", "DATABASE_URL") || fileContains(dir, ".env.local", "DATABASE_URL") {
		nudges = appendUnique(nudges, "supabase")
	}

	// Prisma ORM.
	if fileExists(dir, "prisma/schema.prisma") {
		nudges = appendUnique(nudges, "supabase")
	}

	// Drizzle ORM.
	if fileExists(dir, "drizzle.config.ts") || fileExists(dir, "drizzle.config.js") {
		nudges = appendUnique(nudges, "supabase")
	}

	// Postgres client libraries in package.json.
	if hasAnyDep(dir, "pg", "pgx", "postgres", "psycopg2", "sqlalchemy") {
		nudges = appendUnique(nudges, "supabase")
	}

	// SQL migration files.
	if hasFilesWithExt(dir, ".sql") {
		nudges = appendUnique(nudges, "supabase")
	}
	if hasFilesWithExt(filepath.Join(dir, "migrations"), ".sql") {
		nudges = appendUnique(nudges, "supabase")
	}

	return nudges
}

// --- Helpers ---

func containsAny(slice []string, vals ...string) bool {
	for _, v := range vals {
		if contains(slice, v) {
			return true
		}
	}
	return false
}

func contains(slice []string, val string) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func fileExists(dir, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))
	return err == nil && !info.IsDir()
}

func dirExists(dir, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))
	return err == nil && info.IsDir()
}

func appendUnique(slice []string, val string) []string {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}

// hasDep checks if package.json has a dependency (in dependencies or devDependencies).
func hasDep(dir, dep string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}
	if _, ok := pkg.Dependencies[dep]; ok {
		return true
	}
	if _, ok := pkg.DevDependencies[dep]; ok {
		return true
	}
	return false
}

// hasAnyDep checks if package.json has any of the given dependencies.
func hasAnyDep(dir string, deps ...string) bool {
	for _, dep := range deps {
		if hasDep(dir, dep) {
			return true
		}
	}
	return false
}

// fileContains checks if a file exists and contains the given substring.
func fileContains(dir, name, substr string) bool {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// hasFilesWithExt checks if any files with the given extension exist in dir (non-recursive).
func hasFilesWithExt(dir, ext string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ext) {
			return true
		}
	}
	return false
}
