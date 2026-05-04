# Multi-Framework Detection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the single-match framework detection with accumulative multi-framework detection, add Supabase nudging, and update all consumers.

**Architecture:** Detection becomes a pipeline of independent checks (meta-frameworks, libraries, languages, build indicators, infrastructure) that all fire and accumulate into `Frameworks []string`, `Providers []string`, and `Nudges []string`. Meta-frameworks suppress their underlying libraries (Next.js suppresses React). Nudge logic scans for database indicators when Supabase isn't detected.

**Tech Stack:** Go 1.26.2, Cobra CLI, standard library only (no new deps)

---

### Task 1: Update DetectedStack struct and helpers

**Files:**
- Modify: `internal/detect/detect.go:9-13`

- [ ] **Step 1: Write failing test for new struct shape**

Add to `internal/detect/detect_test.go` at the top, below the existing imports:

```go
func TestDetectStackReturnsFrameworksSlice(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nextjs")
	assertContains(t, stack.Providers, "vercel")
	if len(stack.Nudges) != 0 {
		t.Errorf("expected no nudges, got %v", stack.Nudges)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/detect/ -run TestDetectStackReturnsFrameworksSlice -v`
Expected: FAIL — `stack.Frameworks undefined (type DetectedStack has no field or method Frameworks)`

- [ ] **Step 3: Update the DetectedStack struct**

Replace the struct in `internal/detect/detect.go:9-13` with:

```go
// DetectedStack holds the result of scanning a project directory.
type DetectedStack struct {
	Frameworks []string `json:"frameworks"`
	Providers  []string `json:"providers"`
	Nudges     []string `json:"nudges,omitempty"`
}
```

Remove the `hasReactDep` and `hasExpoDep` functions — they'll be replaced by the generic `hasDep` helper. Remove `isExpoProject` too.

Add these helpers at the bottom of the file (replacing the old dep-check functions):

```go
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
```

Add `"strings"` to the imports.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/detect/ -run TestDetectStackReturnsFrameworksSlice -v`
Expected: FAIL — the `DetectStack` function still sets `Framework` (singular). That's expected; we fix it in Task 2.

- [ ] **Step 5: Commit**

```bash
git add internal/detect/detect.go internal/detect/detect_test.go
git commit -m "refactor: update DetectedStack to use Frameworks slice and Nudges field"
```

---

### Task 2: Rewrite DetectStack with accumulative detection

**Files:**
- Modify: `internal/detect/detect.go:16-67`

- [ ] **Step 1: Write failing tests for multi-framework and new languages**

Replace the entire `internal/detect/detect_test.go` with:

```go
package detect

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Meta-framework tests ---

func TestDetectNextJS(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nextjs")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectNextJSTS(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.ts", "export default {}")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nextjs")
}

func TestDetectNuxt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nuxt.config.ts", "export default {}")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nuxt")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectSvelteKit(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "svelte.config.js", "export default {}")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "sveltekit")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectAstro(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "astro.config.mjs", "export default {}")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "astro")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectRemixConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "remix.config.js", "module.exports = {}")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "remix")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectRemixDep(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"@remix-run/node":"^2.0"}}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "remix")
}

func TestDetectGatsby(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "gatsby-config.js", "module.exports = {}")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "gatsby")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectAngular(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "angular.json", "{}")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "angular")
	assertContains(t, stack.Providers, "vercel")
}

// --- Library tests ---

func TestDetectReact(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.0.0"}}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "react")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectVue(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"vue":"^3.0"}}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "vue")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectSvelte(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"svelte":"^4.0"}}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "svelte")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectSolid(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"solid-js":"^1.0"}}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "solid")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectExpo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"expo":"^50.0"}}`)
	writeFile(t, dir, "app.json", `{"expo":{}}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "expo")
	assertContains(t, stack.Providers, "expo")
}

// --- Meta-framework de-duplication ---

func TestNextJSSuppressesReact(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.0.0"}}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nextjs")
	assertNotContains(t, stack.Frameworks, "react")
}

func TestNuxtSuppressesVue(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nuxt.config.ts", "export default {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"vue":"^3.0"}}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nuxt")
	assertNotContains(t, stack.Frameworks, "vue")
}

func TestSvelteKitSuppressesSvelte(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "svelte.config.js", "export default {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"svelte":"^4.0"}}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "sveltekit")
	assertNotContains(t, stack.Frameworks, "svelte")
}

func TestRemixSuppressesReact(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "remix.config.js", "module.exports = {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.0","@remix-run/node":"^2.0"}}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "remix")
	assertNotContains(t, stack.Frameworks, "react")
}

func TestGatsbySuppressesReact(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "gatsby-config.js", "module.exports = {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.0"}}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "gatsby")
	assertNotContains(t, stack.Frameworks, "react")
}

// --- Language tests ---

func TestDetectGo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "go")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectRust(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "rust")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectPython(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==2.0")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "python")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectPythonPyproject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "python")
}

func TestDetectRuby(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Gemfile", "source 'https://rubygems.org'")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "ruby")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectPHP(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", "{}")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "php")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectJavaMaven(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", "<project/>")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "java")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectJavaGradle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle", "apply plugin: 'java'")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "java")
}

func TestDetectJavaGradleKts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle.kts", "plugins { java }")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "java")
}

func TestDetectScala(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.sbt", "name := \"app\"")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "scala")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectDeno(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "deno.json", "{}")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "deno")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectBun(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bunfig.toml", "[install]")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "bun")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectNodeFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"my-app"}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "node")
	assertContains(t, stack.Providers, "vercel")
}

func TestNodeFallbackSuppressedByFramework(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18"}}`)

	stack := DetectStack(dir)
	assertNotContains(t, stack.Frameworks, "node")
}

// --- Build/deploy indicator tests ---

func TestDetectDocker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Dockerfile", "FROM node:18")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "docker")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectVercelJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "vercel.json", `{"version": 2}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Providers, "vercel")
	if len(stack.Frameworks) != 0 {
		t.Errorf("expected no frameworks for bare vercel.json, got %v", stack.Frameworks)
	}
}

func TestDetectStaticHTML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.html", "<html></html>")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "static")
	assertContains(t, stack.Providers, "vercel")
}

func TestStaticSuppressedByFramework(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	writeFile(t, dir, "index.html", "<html></html>")

	stack := DetectStack(dir)
	assertNotContains(t, stack.Frameworks, "static")
}

// --- Infrastructure tests ---

func TestDetectSupabaseDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	mkdir(t, dir, "supabase")

	stack := DetectStack(dir)
	assertContains(t, stack.Providers, "vercel")
	assertContains(t, stack.Providers, "supabase")
}

func TestDetectEasJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "eas.json", "{}")

	stack := DetectStack(dir)
	assertContains(t, stack.Providers, "expo")
}

// --- Multi-framework (monorepo) tests ---

func TestMonorepoNextJSPlusGo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18"}}`)
	writeFile(t, dir, "go.mod", "module example.com/app")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nextjs")
	assertContains(t, stack.Frameworks, "go")
	assertNotContains(t, stack.Frameworks, "react")
	assertContains(t, stack.Providers, "vercel")
}

func TestMonorepoNextJSPlusGoPlusSupabase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18"}}`)
	writeFile(t, dir, "go.mod", "module example.com/app")
	mkdir(t, dir, "supabase")

	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nextjs")
	assertContains(t, stack.Frameworks, "go")
	assertContains(t, stack.Providers, "vercel")
	assertContains(t, stack.Providers, "supabase")
}

// --- Edge cases ---

func TestDetectEmpty(t *testing.T) {
	dir := t.TempDir()

	stack := DetectStack(dir)
	if len(stack.Frameworks) != 0 {
		t.Errorf("expected no frameworks, got %v", stack.Frameworks)
	}
	if len(stack.Providers) != 0 {
		t.Errorf("expected no providers, got %v", stack.Providers)
	}
	if len(stack.Nudges) != 0 {
		t.Errorf("expected no nudges, got %v", stack.Nudges)
	}
}

// --- helpers ---

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, name), 0755); err != nil {
		t.Fatal(err)
	}
}

func assertContains(t *testing.T, slice []string, val string) {
	t.Helper()
	for _, v := range slice {
		if v == val {
			return
		}
	}
	t.Errorf("expected %v to contain %q", slice, val)
}

func assertNotContains(t *testing.T, slice []string, val string) {
	t.Helper()
	for _, v := range slice {
		if v == val {
			t.Errorf("expected %v NOT to contain %q", slice, val)
			return
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/detect/ -v 2>&1 | head -30`
Expected: Multiple failures — `DetectStack` still uses old logic.

- [ ] **Step 3: Rewrite DetectStack**

Replace the entire `DetectStack` function in `internal/detect/detect.go` with:

```go
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

// containsAny returns true if slice contains any of the given values.
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
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/detect/ -v 2>&1 | tail -5`
Expected: Most tests pass. Nudge tests will fail (Task 3).

- [ ] **Step 5: Commit**

```bash
git add internal/detect/detect.go internal/detect/detect_test.go
git commit -m "feat: rewrite DetectStack with accumulative multi-framework detection"
```

---

### Task 3: Implement Supabase nudge detection

**Files:**
- Modify: `internal/detect/detect.go` (add `detectSupabaseNudges`)
- Modify: `internal/detect/detect_test.go` (add nudge tests)

- [ ] **Step 1: Write failing nudge tests**

Add to `internal/detect/detect_test.go` before the helpers section:

```go
// --- Supabase nudge tests ---

func TestNudgeDockerComposePostgres(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app")
	writeFile(t, dir, "docker-compose.yml", "services:\n  db:\n    image: postgres:15")

	stack := DetectStack(dir)
	assertContains(t, stack.Nudges, "supabase")
}

func TestNudgeEnvDatabaseURL(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app")
	writeFile(t, dir, ".env", "DATABASE_URL=postgres://localhost/mydb")

	stack := DetectStack(dir)
	assertContains(t, stack.Nudges, "supabase")
}

func TestNudgePrisma(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18"}}`)
	mkdir(t, dir, "prisma")
	writeFile(t, dir, "prisma/schema.prisma", "generator client {}")

	stack := DetectStack(dir)
	assertContains(t, stack.Nudges, "supabase")
}

func TestNudgeDrizzle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18"}}`)
	writeFile(t, dir, "drizzle.config.ts", "export default {}")

	stack := DetectStack(dir)
	assertContains(t, stack.Nudges, "supabase")
}

func TestNudgePgDep(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"pg":"^8.0"}}`)

	stack := DetectStack(dir)
	assertContains(t, stack.Nudges, "supabase")
}

func TestNudgeSQLFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app")
	mkdir(t, dir, "migrations")
	writeFile(t, dir, "migrations/001_init.sql", "CREATE TABLE users();")

	stack := DetectStack(dir)
	assertContains(t, stack.Nudges, "supabase")
}

func TestNoNudgeWhenSupabaseExists(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app")
	writeFile(t, dir, "docker-compose.yml", "services:\n  db:\n    image: postgres:15")
	mkdir(t, dir, "supabase")

	stack := DetectStack(dir)
	if len(stack.Nudges) != 0 {
		t.Errorf("expected no nudges when supabase/ exists, got %v", stack.Nudges)
	}
}

func TestNoNudgeWhenNoDatabaseIndicators(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app")

	stack := DetectStack(dir)
	if len(stack.Nudges) != 0 {
		t.Errorf("expected no nudges, got %v", stack.Nudges)
	}
}
```

- [ ] **Step 2: Run tests to verify nudge tests fail**

Run: `go test ./internal/detect/ -run TestNudge -v`
Expected: FAIL — `detectSupabaseNudges` not defined.

- [ ] **Step 3: Implement detectSupabaseNudges**

Add to `internal/detect/detect.go`:

```go
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
```

- [ ] **Step 4: Run all detection tests**

Run: `go test ./internal/detect/ -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/detect/detect.go internal/detect/detect_test.go
git commit -m "feat: add Supabase nudge detection for database indicators"
```

---

### Task 4: Update cmd/doctor.go for new struct

**Files:**
- Modify: `cmd/doctor.go:136`

- [ ] **Step 1: Update Framework reference to Frameworks**

In `cmd/doctor.go`, change line 136 from:

```go
"framework":       projCfg.DetectedStack.Framework,
```

to:

```go
"frameworks":      projCfg.DetectedStack.Frameworks,
```

- [ ] **Step 2: Add nudge surfacing**

After the cross-provider connectivity check block (around line 131), add:

```go
	// Surface Supabase nudges.
	if len(projCfg.DetectedStack.Nudges) > 0 {
		for _, nudge := range projCfg.DetectedStack.Nudges {
			if nudge == "supabase" {
				warnings = append(warnings, "This project has database indicators but is not using Supabase. Supabase provides managed Postgres, authentication, edge functions, and real-time subscriptions. To adopt Supabase, run 'supabase init' to create the supabase/ directory, then re-run 'vibecloud init'.")
			}
		}
	}
```

Also add `"nudges"` to the data map:

```go
	if len(projCfg.DetectedStack.Nudges) > 0 {
		data["nudges"] = projCfg.DetectedStack.Nudges
	}
```

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: Compiles cleanly.

- [ ] **Step 4: Commit**

```bash
git add cmd/doctor.go
git commit -m "feat: update doctor for multi-framework detection and nudges"
```

---

### Task 5: Update cmd/init.go for new struct

**Files:**
- Modify: `cmd/init.go:167-169`
- Modify: `cmd/init.go:252` (writeClaudeMD)

- [ ] **Step 1: Update Framework reference to Frameworks**

In `cmd/init.go`, change the instructions line (around line 167-169) from:

```go
	instructions := fmt.Sprintf(
		"Project '%s' initialized as %s with providers [%s]. ",
		projectName, stack.Framework, strings.Join(stack.Providers, ", "),
	)
```

to:

```go
	instructions := fmt.Sprintf(
		"Project '%s' initialized as %s with providers [%s]. ",
		projectName, strings.Join(stack.Frameworks, ", "), strings.Join(stack.Providers, ", "),
	)
```

- [ ] **Step 2: Add nudge to init instructions**

After the instructions block (after line 175), add:

```go
	if len(stack.Nudges) > 0 {
		for _, nudge := range stack.Nudges {
			if nudge == "supabase" {
				instructions += " Note: This project has database indicators but is not using Supabase. Supabase provides managed Postgres, auth, edge functions, and real-time. Run 'supabase init' then re-run 'vibecloud init' to add Supabase."
			}
		}
	}
```

- [ ] **Step 3: Update writeClaudeMD to use Frameworks**

In the `writeClaudeMD` function, the `vibeSection` template uses `stack.Providers` for the provider list but references `stack.Framework` nowhere directly (it was already using `strings.Join(stack.Providers, ...)`). No change needed in the template itself.

However, verify by reading the function — the two `%s` format verbs reference `strings.Join(stack.Providers, "/")` and `strings.Join(stack.Providers, ", ")`. These are already correct since they reference `Providers`, not `Framework`.

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: Compiles cleanly.

- [ ] **Step 5: Commit**

```bash
git add cmd/init.go
git commit -m "feat: update init for multi-framework detection and nudges"
```

---

### Task 6: Update cmd/explain.go for new struct

**Files:**
- Modify: `cmd/explain.go:81,86-89`

- [ ] **Step 1: Update Framework references to Frameworks**

In `cmd/explain.go`, change line 81 from:

```go
		"framework": projCfg.DetectedStack.Framework,
```

to:

```go
		"frameworks": projCfg.DetectedStack.Frameworks,
```

Change lines 86-89 from:

```go
	instructions := fmt.Sprintf(
		"Project '%s' (%s). %s. Use 'vibecloud doctor' to check deploy-readiness or 'vibecloud deploy' to deploy.",
		projCfg.ProjectName,
		projCfg.DetectedStack.Framework,
		strings.Join(summaryParts, ". "),
	)
```

to:

```go
	instructions := fmt.Sprintf(
		"Project '%s' (%s). %s. Use 'vibecloud doctor' to check deploy-readiness or 'vibecloud deploy' to deploy.",
		projCfg.ProjectName,
		strings.Join(projCfg.DetectedStack.Frameworks, ", "),
		strings.Join(summaryParts, ". "),
	)
```

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Compiles cleanly.

- [ ] **Step 3: Commit**

```bash
git add cmd/explain.go
git commit -m "feat: update explain for multi-framework detection"
```

---

### Task 7: Final verification

**Files:** None (read-only verification)

- [ ] **Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass.

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: Clean.

- [ ] **Step 3: Build binary**

Run: `go build -o /dev/null .`
Expected: Compiles cleanly.

- [ ] **Step 4: Verify .vibecloud.json schema change**

Run: `grep -r "Framework " cmd/ internal/ --include="*.go" | grep -v "_test.go" | grep -v "Frameworks"`
Expected: No results — all references should be `Frameworks` (plural).

- [ ] **Step 5: Commit any remaining changes**

```bash
git add -A
git status
# If clean, nothing to commit. Otherwise:
git commit -m "chore: final cleanup for multi-framework detection"
```
